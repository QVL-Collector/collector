package stockage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"collector/libs/platform/outbox"
	"collector/services/place-de-marche/internal/annonces"
)

const articleColumns = `a.id, a.vendeur_id, a.etat, a.revision, a.titre, a.description,
	a.categorie_id, c.libelle, a.prix_centimes, a.frais_port_centimes, a.motif, a.cree_le, a.modifie_le`

const insertDraftQuery = `
WITH cree AS (
	INSERT INTO article (vendeur_id, etat, revision, titre, description, categorie_id, prix_centimes, frais_port_centimes)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING *
)
SELECT ` + articleColumns + `
FROM cree AS a LEFT JOIN categorie AS c ON c.id = a.categorie_id`

const memberSinceColumn = `(SELECT (min(premier.cree_le) AT TIME ZONE 'UTC')::date FROM article AS premier WHERE premier.vendeur_id = a.vendeur_id)`

const contentAssignments = `titre = $5, description = $6, categorie_id = $7, prix_centimes = $8, frais_port_centimes = $9`

const updateDraftQuery = `
WITH modifie AS (
	UPDATE article
	SET ` + contentAssignments + `, revision = revision + 1, modifie_le = now()
	WHERE id = $1 AND vendeur_id = $2 AND revision = $3 AND etat = ANY($4)
	RETURNING *
)
SELECT ` + articleColumns + `
FROM modifie AS a LEFT JOIN categorie AS c ON c.id = a.categorie_id`

const reviseQuery = `
WITH revise AS (
	UPDATE article
	SET ` + contentAssignments + `, etat = $10, motif = NULL, revision = revision + 1, modifie_le = now()
	WHERE id = $1 AND vendeur_id = $2 AND revision = $3 AND etat = ANY($4)
	RETURNING *
)
SELECT ` + articleColumns + `,
	` + memberSinceColumn + `
FROM revise AS a LEFT JOIN categorie AS c ON c.id = a.categorie_id`

const submitQuery = `
WITH soumis AS (
	UPDATE article
	SET etat = $5, motif = NULL, revision = revision + 1, modifie_le = now()
	WHERE id = $1 AND vendeur_id = $2 AND revision = $3 AND etat = ANY($4)
	RETURNING *
)
SELECT ` + articleColumns + `,
	` + memberSinceColumn + `
FROM soumis AS a LEFT JOIN categorie AS c ON c.id = a.categorie_id`

const applyReviewQuery = `
UPDATE article
SET etat = $3, motif = $4, modifie_le = now()
WHERE id = $1 AND revision = $2 AND etat = $5`

const recordReviewQuery = `
INSERT INTO resultat_controle (message_id, article_id, revision, issue, motif, version_regles)
VALUES ($1::text::uuid, $2, $3, $4, $5, $6)`

const findOwnedQuery = `
SELECT ` + articleColumns + `
FROM article AS a LEFT JOIN categorie AS c ON c.id = a.categorie_id
WHERE a.id = $1 AND a.vendeur_id = $2`

const listOwnedQuery = `
SELECT ` + articleColumns + `
FROM article AS a LEFT JOIN categorie AS c ON c.id = a.categorie_id
WHERE a.vendeur_id = $1
ORDER BY a.modifie_le DESC, a.id DESC
LIMIT $2 OFFSET $3`

const (
	countOwnedQuery     = `SELECT count(*) FROM article WHERE vendeur_id = $1`
	ownedStateQuery     = `SELECT etat, revision FROM article WHERE id = $1 AND vendeur_id = $2`
	lockedStateQuery    = ownedStateQuery + ` FOR UPDATE`
	listCategoriesQuery = `SELECT id, libelle FROM categorie ORDER BY libelle, id`
	categoryExistsQuery = `SELECT EXISTS (SELECT 1 FROM categorie WHERE id = $1)`
)

type ReviewRequestEncoder interface {
	Encode(ctx context.Context, request annonces.ReviewRequest) (outbox.Event, error)
}

type Store struct {
	pool     *pgxpool.Pool
	requests ReviewRequestEncoder
}

func NewStore(pool *pgxpool.Pool, requests ReviewRequestEncoder) *Store {
	return &Store{pool: pool, requests: requests}
}

func (s *Store) CreateDraft(ctx context.Context, seller annonces.SellerID, content annonces.Content) (annonces.Article, error) {
	columns := contentColumns(content)
	row := s.pool.QueryRow(ctx, insertDraftQuery,
		uuid.UUID(seller), string(annonces.StatusDraft), int(annonces.FirstRevision),
		columns.title, columns.description, columns.categoryID, columns.priceCents, columns.shippingFeeCents,
	)
	article, err := scanArticle(row)
	if err != nil {
		return annonces.Article{}, fmt.Errorf("création du brouillon : %w", err)
	}
	return article, nil
}

func (s *Store) UpdateDraft(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, expected annonces.Revision, content annonces.Content) (annonces.Article, error) {
	var updated annonces.Article
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		status, err := lockEditable(ctx, tx, seller, id, expected)
		if err != nil {
			return err
		}
		if status.ReviewedOnEdit() {
			updated, err = s.reviseInTransaction(ctx, tx, seller, id, expected, content)
			return err
		}
		updated, err = updateDraftInTransaction(ctx, tx, seller, id, expected, content)
		return err
	})
	if err != nil {
		return annonces.Article{}, fmt.Errorf("modification de l'article : %w", err)
	}
	return updated, nil
}

func lockEditable(ctx context.Context, tx pgx.Tx, seller annonces.SellerID, id annonces.ArticleID, expected annonces.Revision) (annonces.Status, error) {
	status, current, err := ownedState(ctx, tx, lockedStateQuery, seller, id)
	if err != nil {
		return "", err
	}
	return status, annonces.VerifyEditable(status, current, expected)
}

func updateDraftInTransaction(ctx context.Context, tx pgx.Tx, seller annonces.SellerID, id annonces.ArticleID, expected annonces.Revision, content annonces.Content) (annonces.Article, error) {
	columns := contentColumns(content)
	row := tx.QueryRow(ctx, updateDraftQuery,
		uuid.UUID(id), uuid.UUID(seller), int(expected), statusColumns(annonces.SellerHeldStatuses()),
		columns.title, columns.description, columns.categoryID, columns.priceCents, columns.shippingFeeCents,
	)
	return scanArticle(row)
}

func (s *Store) reviseInTransaction(ctx context.Context, tx pgx.Tx, seller annonces.SellerID, id annonces.ArticleID, expected annonces.Revision, content annonces.Content) (annonces.Article, error) {
	columns := contentColumns(content)
	row := tx.QueryRow(ctx, reviseQuery,
		uuid.UUID(id), uuid.UUID(seller), int(expected), statusColumns(annonces.ReviewOnEditStatuses()),
		columns.title, columns.description, columns.categoryID, columns.priceCents, columns.shippingFeeCents,
		string(annonces.StatusUnderReview),
	)
	request, err := scanReviewRequest(row)
	if err != nil {
		return annonces.Article{}, err
	}
	return request.Article, s.enqueueReview(ctx, tx, request)
}

func (s *Store) Submit(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, expected annonces.Revision) (annonces.Article, error) {
	var submitted annonces.Article
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		request, err := submitInTransaction(ctx, tx, seller, id, expected)
		if err != nil {
			return err
		}
		submitted = request.Article
		return s.enqueueReview(ctx, tx, request)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return annonces.Article{}, s.explainRefusal(ctx, seller, id, annonces.Status.Submittable, annonces.ErrArticleNotSubmittable)
	}
	if err != nil {
		return annonces.Article{}, fmt.Errorf("soumission de l'article : %w", err)
	}
	return submitted, nil
}

func submitInTransaction(ctx context.Context, tx pgx.Tx, seller annonces.SellerID, id annonces.ArticleID, expected annonces.Revision) (annonces.ReviewRequest, error) {
	row := tx.QueryRow(ctx, submitQuery,
		uuid.UUID(id), uuid.UUID(seller), int(expected), statusColumns(annonces.SellerHeldStatuses()), string(annonces.StatusUnderReview),
	)
	return scanReviewRequest(row)
}

func scanReviewRequest(row pgx.Row) (annonces.ReviewRequest, error) {
	var (
		r           articleRow
		memberSince time.Time
	)
	if err := row.Scan(append(r.targets(), &memberSince)...); err != nil {
		return annonces.ReviewRequest{}, err
	}
	article := r.article()
	return annonces.ReviewRequest{
		Article:     article,
		Seller:      annonces.SellerStatus{MemberSince: memberSince},
		SubmittedAt: article.UpdatedAt,
	}, nil
}

func (s *Store) enqueueReview(ctx context.Context, tx pgx.Tx, request annonces.ReviewRequest) error {
	event, err := s.requests.Encode(ctx, request)
	if err != nil {
		return err
	}
	return outbox.Enqueue(ctx, tx, event)
}

func (s *Store) ApplyReview(ctx context.Context, outcome annonces.ReviewOutcome) (bool, error) {
	applied := false
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		reason := reasonColumn(outcome.Reason)
		tag, err := tx.Exec(ctx, applyReviewQuery,
			uuid.UUID(outcome.Article), int(outcome.Revision), string(outcome.Status), reason, string(annonces.StatusUnderReview),
		)
		if err != nil || tag.RowsAffected() == 0 {
			return err
		}
		applied = true
		_, err = tx.Exec(ctx, recordReviewQuery,
			outcome.MessageID, uuid.UUID(outcome.Article), int(outcome.Revision), string(outcome.Status), reason, outcome.RulesVersion,
		)
		return err
	})
	if err != nil {
		return false, fmt.Errorf("application du résultat de contrôle : %w", err)
	}
	return applied, nil
}

func (s *Store) explainRefusal(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, allowed func(annonces.Status) bool, notAllowed error) error {
	status, _, err := ownedState(ctx, s.pool, ownedStateQuery, seller, id)
	if err != nil {
		return err
	}
	if !allowed(status) {
		return notAllowed
	}
	return annonces.ErrStaleRevision
}

type rowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func ownedState(ctx context.Context, querier rowQuerier, query string, seller annonces.SellerID, id annonces.ArticleID) (annonces.Status, annonces.Revision, error) {
	var (
		status   string
		revision int
	)
	err := querier.QueryRow(ctx, query, uuid.UUID(id), uuid.UUID(seller)).Scan(&status, &revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, annonces.ErrArticleNotFound
	}
	if err != nil {
		return "", 0, fmt.Errorf("lecture de l'état de l'article : %w", err)
	}
	return annonces.Status(status), annonces.Revision(revision), nil
}

func (s *Store) FindOwned(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID) (annonces.Article, error) {
	article, err := scanArticle(s.pool.QueryRow(ctx, findOwnedQuery, uuid.UUID(id), uuid.UUID(seller)))
	if errors.Is(err, pgx.ErrNoRows) {
		return annonces.Article{}, annonces.ErrArticleNotFound
	}
	if err != nil {
		return annonces.Article{}, fmt.Errorf("lecture de l'article : %w", err)
	}
	return article, nil
}

func (s *Store) ListOwned(ctx context.Context, seller annonces.SellerID, page annonces.PageRequest) (annonces.ArticlePage, error) {
	var total int
	if err := s.pool.QueryRow(ctx, countOwnedQuery, uuid.UUID(seller)).Scan(&total); err != nil {
		return annonces.ArticlePage{}, fmt.Errorf("décompte des articles : %w", err)
	}
	rows, err := s.pool.Query(ctx, listOwnedQuery, uuid.UUID(seller), page.Size, page.Offset())
	if err != nil {
		return annonces.ArticlePage{}, fmt.Errorf("liste des articles : %w", err)
	}
	articles, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (annonces.Article, error) {
		return scanArticle(row)
	})
	if err != nil {
		return annonces.ArticlePage{}, fmt.Errorf("lecture des articles : %w", err)
	}
	return annonces.ArticlePage{Articles: articles, Total: total}, nil
}

func (s *Store) ListCategories(ctx context.Context) ([]annonces.Category, error) {
	rows, err := s.pool.Query(ctx, listCategoriesQuery)
	if err != nil {
		return nil, fmt.Errorf("liste des catégories : %w", err)
	}
	categories, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (annonces.Category, error) {
		var id, label string
		err := row.Scan(&id, &label)
		return annonces.Category{ID: annonces.CategoryID(id), Label: label}, err
	})
	if err != nil {
		return nil, fmt.Errorf("lecture des catégories : %w", err)
	}
	return categories, nil
}

func (s *Store) CategoryExists(ctx context.Context, id annonces.CategoryID) (bool, error) {
	var exists bool
	if err := s.pool.QueryRow(ctx, categoryExistsQuery, string(id)).Scan(&exists); err != nil {
		return false, fmt.Errorf("recherche de la catégorie : %w", err)
	}
	return exists, nil
}

type articleRow struct {
	id               uuid.UUID
	seller           uuid.UUID
	status           string
	revision         int
	title            string
	description      string
	categoryID       *string
	categoryLabel    *string
	priceCents       *int64
	shippingFeeCents *int64
	reason           *string
	createdAt        time.Time
	updatedAt        time.Time
}

func (r *articleRow) targets() []any {
	return []any{&r.id, &r.seller, &r.status, &r.revision, &r.title, &r.description,
		&r.categoryID, &r.categoryLabel, &r.priceCents, &r.shippingFeeCents, &r.reason, &r.createdAt, &r.updatedAt}
}

func scanArticle(row pgx.Row) (annonces.Article, error) {
	var r articleRow
	if err := row.Scan(r.targets()...); err != nil {
		return annonces.Article{}, err
	}
	return r.article(), nil
}

func (r articleRow) article() annonces.Article {
	return annonces.Article{
		ID:          annonces.ArticleID(r.id),
		Seller:      annonces.SellerID(r.seller),
		Status:      annonces.Status(r.status),
		Revision:    annonces.Revision(r.revision),
		Title:       r.title,
		Description: r.description,
		Category:    category(r.categoryID, r.categoryLabel),
		Price:       euros(r.priceCents),
		ShippingFee: euros(r.shippingFeeCents),
		Reason:      reviewReason(r.reason),
		CreatedAt:   r.createdAt,
		UpdatedAt:   r.updatedAt,
	}
}

func category(id, label *string) *annonces.Category {
	if id == nil || label == nil {
		return nil
	}
	return &annonces.Category{ID: annonces.CategoryID(*id), Label: *label}
}

func euros(cents *int64) *annonces.Money {
	if cents == nil {
		return nil
	}
	return &annonces.Money{Cents: *cents, Currency: annonces.Euro}
}

func reviewReason(raw *string) *annonces.ReviewReason {
	if raw == nil {
		return nil
	}
	reason := annonces.ReviewReason(*raw)
	return &reason
}

func reasonColumn(reason *annonces.ReviewReason) *string {
	if reason == nil {
		return nil
	}
	value := string(*reason)
	return &value
}

func statusColumns(statuses []annonces.Status) []string {
	columns := make([]string, len(statuses))
	for i, status := range statuses {
		columns[i] = string(status)
	}
	return columns
}

type draftColumns struct {
	title            string
	description      string
	categoryID       *string
	priceCents       *int64
	shippingFeeCents *int64
}

func contentColumns(content annonces.Content) draftColumns {
	return draftColumns{
		title:            content.Title,
		description:      content.Description,
		categoryID:       categoryColumn(content.CategoryID),
		priceCents:       centsColumn(content.Price),
		shippingFeeCents: centsColumn(content.ShippingFee),
	}
}

func categoryColumn(id *annonces.CategoryID) *string {
	if id == nil {
		return nil
	}
	value := string(*id)
	return &value
}

func centsColumn(money *annonces.Money) *int64 {
	if money == nil {
		return nil
	}
	cents := money.Cents
	return &cents
}
