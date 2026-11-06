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
	"collector/services/place-de-marche/internal/comptes"
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

const reviewRequestedAssignments = `demande_controle_le = now(), relances = 0`

const reviseQuery = `
WITH revise AS (
	UPDATE article
	SET ` + contentAssignments + `, etat = $10, motif = NULL, revision = revision + 1, modifie_le = now(), ` + reviewRequestedAssignments + `
	WHERE id = $1 AND vendeur_id = $2 AND revision = $3 AND etat = ANY($4)
	RETURNING *
)
SELECT ` + articleColumns + `
FROM revise AS a LEFT JOIN categorie AS c ON c.id = a.categorie_id`

const submitQuery = `
WITH soumis AS (
	UPDATE article
	SET etat = $5, motif = NULL, revision = revision + 1, modifie_le = now(), ` + reviewRequestedAssignments + `
	WHERE id = $1 AND vendeur_id = $2 AND revision = $3 AND etat = ANY($4)
	RETURNING *
)
SELECT ` + articleColumns + `
FROM soumis AS a LEFT JOIN categorie AS c ON c.id = a.categorie_id`

const resendDelaySeconds = `least($2 * power(2, least(a.relances, 30)), $3)`

const resendOverdueQuery = `
WITH en_retard AS (
	SELECT a.id
	FROM article AS a
	WHERE a.etat = $1 AND a.demande_controle_le + make_interval(secs => ` + resendDelaySeconds + `) <= now()
	ORDER BY a.demande_controle_le, a.id
	LIMIT $4
	FOR UPDATE SKIP LOCKED
),
relance AS (
	UPDATE article AS a
	SET demande_controle_le = now(), relances = a.relances + 1
	FROM en_retard
	WHERE a.id = en_retard.id
	RETURNING a.*
)
SELECT ` + articleColumns + `, a.relances, ` + resendDelaySeconds + `
FROM relance AS a LEFT JOIN categorie AS c ON c.id = a.categorie_id`

const applyReviewQuery = `
UPDATE article
SET etat = $3, motif = $4, modifie_le = now()
WHERE id = $1 AND revision = $2 AND etat = $5
RETURNING vendeur_id, modifie_le`

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

const articlesToVerify = `
FROM article AS a LEFT JOIN categorie AS c ON c.id = a.categorie_id
WHERE a.etat = 'a_verifier'`

const (
	countToVerifyQuery = `SELECT count(*)` + articlesToVerify
	listToVerifyQuery  = `SELECT ` + articleColumns + articlesToVerify + `
ORDER BY a.modifie_le ASC, a.id ASC
LIMIT $1 OFFSET $2`
)

const decideQuery = `
WITH tranche AS (
	UPDATE article
	SET etat = $4, motif = $5, modifie_le = now()
	WHERE id = $1 AND revision = $2 AND etat = $3
	RETURNING *
)
SELECT ` + articleColumns + `
FROM tranche AS a LEFT JOIN categorie AS c ON c.id = a.categorie_id`

const recordDecisionQuery = `
INSERT INTO decision_admin (article_id, revision, decision, motif, admin_id)
VALUES ($1, $2, $3, $4, $5)`

const (
	articleStateQuery   = `SELECT etat, revision FROM article WHERE id = $1`
	countOwnedQuery     = `SELECT count(*) FROM article WHERE vendeur_id = $1`
	ownedStateQuery     = `SELECT etat, revision FROM article WHERE id = $1 AND vendeur_id = $2`
	lockedStateQuery    = ownedStateQuery + ` FOR UPDATE`
	listCategoriesQuery = `SELECT id, libelle FROM categorie ORDER BY libelle, id`
	categoryExistsQuery = `SELECT EXISTS (SELECT 1 FROM categorie WHERE id = $1)`
)

type ReviewRequestEncoder interface {
	Encode(ctx context.Context, request annonces.ReviewRequest) (outbox.Event, error)
}

type PublicationDecisionEncoder interface {
	Encode(ctx context.Context, decision annonces.PublicationDecision) (outbox.Event, error)
}

type SellerAccounts interface {
	Enroll(ctx context.Context, tx pgx.Tx, id comptes.AccountID) error
	Status(ctx context.Context, tx pgx.Tx, id comptes.AccountID) (comptes.Status, error)
}

type Store struct {
	pool      *pgxpool.Pool
	requests  ReviewRequestEncoder
	decisions PublicationDecisionEncoder
	accounts  SellerAccounts
}

func NewStore(pool *pgxpool.Pool, requests ReviewRequestEncoder, decisions PublicationDecisionEncoder, accounts SellerAccounts) *Store {
	return &Store{pool: pool, requests: requests, decisions: decisions, accounts: accounts}
}

func (s *Store) CreateDraft(ctx context.Context, seller annonces.SellerID, content annonces.Content) (annonces.Article, error) {
	var created annonces.Article
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := s.accounts.Enroll(ctx, tx, accountOf(seller)); err != nil {
			return err
		}
		var err error
		created, err = insertDraftInTransaction(ctx, tx, seller, content)
		return err
	})
	if err != nil {
		return annonces.Article{}, fmt.Errorf("création du brouillon : %w", err)
	}
	return created, nil
}

func insertDraftInTransaction(ctx context.Context, tx pgx.Tx, seller annonces.SellerID, content annonces.Content) (annonces.Article, error) {
	columns := contentColumns(content)
	row := tx.QueryRow(ctx, insertDraftQuery,
		uuid.UUID(seller), string(annonces.StatusDraft), int(annonces.FirstRevision),
		columns.title, columns.description, columns.categoryID, columns.priceCents, columns.shippingFeeCents,
	)
	return scanArticle(row)
}

func accountOf(seller annonces.SellerID) comptes.AccountID {
	return comptes.AccountID(seller)
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
	revised, err := scanArticle(row)
	if err != nil {
		return annonces.Article{}, err
	}
	return revised, s.requestReview(ctx, tx, revised)
}

func (s *Store) Submit(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, expected annonces.Revision) (annonces.Article, error) {
	var submitted annonces.Article
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		submitted, err = submitInTransaction(ctx, tx, seller, id, expected)
		if err != nil {
			return err
		}
		return s.requestReview(ctx, tx, submitted)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return annonces.Article{}, s.explainRefusal(ctx, seller, id, annonces.Status.Submittable, annonces.ErrArticleNotSubmittable)
	}
	if err != nil {
		return annonces.Article{}, fmt.Errorf("soumission de l'article : %w", err)
	}
	return submitted, nil
}

func submitInTransaction(ctx context.Context, tx pgx.Tx, seller annonces.SellerID, id annonces.ArticleID, expected annonces.Revision) (annonces.Article, error) {
	row := tx.QueryRow(ctx, submitQuery,
		uuid.UUID(id), uuid.UUID(seller), int(expected), statusColumns(annonces.SellerHeldStatuses()), string(annonces.StatusUnderReview),
	)
	return scanArticle(row)
}

func (s *Store) requestReview(ctx context.Context, tx pgx.Tx, article annonces.Article) error {
	seller, err := s.sellerStatus(ctx, tx, article.Seller)
	if err != nil {
		return err
	}
	event, err := s.requests.Encode(ctx, annonces.ReviewRequest{
		Article:     article,
		Seller:      seller,
		SubmittedAt: article.UpdatedAt,
	})
	if err != nil {
		return err
	}
	return outbox.Enqueue(ctx, tx, event)
}

func (s *Store) sellerStatus(ctx context.Context, tx pgx.Tx, seller annonces.SellerID) (annonces.SellerStatus, error) {
	account := accountOf(seller)
	if err := s.accounts.Enroll(ctx, tx, account); err != nil {
		return annonces.SellerStatus{}, err
	}
	status, err := s.accounts.Status(ctx, tx, account)
	if err != nil {
		return annonces.SellerStatus{}, err
	}
	return annonces.SellerStatus{MemberSince: status.RegisteredAt, Suspended: status.Suspended}, nil
}

func (s *Store) ApplyReview(ctx context.Context, outcome annonces.ReviewOutcome) (bool, error) {
	applied := false
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		decision, err := applyReviewInTransaction(ctx, tx, outcome)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if err = recordReview(ctx, tx, outcome); err != nil {
			return err
		}
		if err = s.announceDecision(ctx, tx, decision); err != nil {
			return err
		}
		applied = true
		return nil
	})
	if err != nil {
		return false, fmt.Errorf("application du résultat de contrôle : %w", err)
	}
	return applied, nil
}

func applyReviewInTransaction(ctx context.Context, tx pgx.Tx, outcome annonces.ReviewOutcome) (annonces.PublicationDecision, error) {
	var (
		seller    uuid.UUID
		decidedAt time.Time
	)
	err := tx.QueryRow(ctx, applyReviewQuery,
		uuid.UUID(outcome.Article), int(outcome.Revision), string(outcome.Status), reasonColumn(outcome.Reason), string(annonces.StatusUnderReview),
	).Scan(&seller, &decidedAt)
	if err != nil {
		return annonces.PublicationDecision{}, err
	}
	return outcome.DecisionFor(annonces.SellerID(seller), decidedAt), nil
}

func recordReview(ctx context.Context, tx pgx.Tx, outcome annonces.ReviewOutcome) error {
	_, err := tx.Exec(ctx, recordReviewQuery,
		outcome.MessageID, uuid.UUID(outcome.Article), int(outcome.Revision), string(outcome.Status), reasonColumn(outcome.Reason), outcome.RulesVersion,
	)
	return err
}

func (s *Store) announceDecision(ctx context.Context, tx pgx.Tx, decision annonces.PublicationDecision) error {
	event, err := s.decisions.Encode(ctx, decision)
	if err != nil {
		return err
	}
	return outbox.Enqueue(ctx, tx, event)
}

func (s *Store) ResendOverdueReviews(ctx context.Context, backoff annonces.ResendBackoff, limit int) ([]annonces.ResentReview, error) {
	var resent []annonces.ResentReview
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		resent, err = overdueReviewsInTransaction(ctx, tx, backoff, limit)
		if err != nil {
			return err
		}
		for _, review := range resent {
			if err = s.requestReview(ctx, tx, review.Article); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("relance des contrôles en retard : %w", err)
	}
	return resent, nil
}

func overdueReviewsInTransaction(ctx context.Context, tx pgx.Tx, backoff annonces.ResendBackoff, limit int) ([]annonces.ResentReview, error) {
	rows, err := tx.Query(ctx, resendOverdueQuery,
		string(annonces.StatusUnderReview), backoff.First.Seconds(), backoff.Max.Seconds(), limit,
	)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (annonces.ResentReview, error) {
		return scanResentReview(row)
	})
}

func scanResentReview(row pgx.Row) (annonces.ResentReview, error) {
	var (
		r           articleRow
		resends     int
		nextSeconds float64
	)
	if err := row.Scan(append(r.targets(), &resends, &nextSeconds)...); err != nil {
		return annonces.ResentReview{}, err
	}
	return annonces.ResentReview{
		Article: r.article(),
		Resends: resends,
		NextIn:  time.Duration(nextSeconds * float64(time.Second)),
	}, nil
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

func (s *Store) ListToVerify(ctx context.Context, page annonces.PageRequest) (annonces.ArticlePage, error) {
	var toVerify annonces.ArticlePage
	err := pgx.BeginTxFunc(ctx, s.pool, readOnlySnapshot, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, countToVerifyQuery).Scan(&toVerify.Total); err != nil {
			return fmt.Errorf("décompte des articles à vérifier : %w", err)
		}
		rows, err := tx.Query(ctx, listToVerifyQuery, page.Size, page.Offset())
		if err != nil {
			return fmt.Errorf("liste des articles à vérifier : %w", err)
		}
		toVerify.Articles, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (annonces.Article, error) {
			return scanArticle(row)
		})
		return err
	})
	if err != nil {
		return annonces.ArticlePage{}, fmt.Errorf("lecture des articles à vérifier : %w", err)
	}
	return toVerify, nil
}

func (s *Store) Decide(ctx context.Context, decision annonces.Decision) (annonces.Article, error) {
	var decided annonces.Article
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		decided, err = decideInTransaction(ctx, tx, decision)
		if err != nil {
			return err
		}
		if err = recordDecision(ctx, tx, decision); err != nil {
			return err
		}
		return s.announceDecision(ctx, tx, annonces.DecisionOn(decided))
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return annonces.Article{}, s.explainDecisionConflict(ctx, decision)
	}
	if err != nil {
		return annonces.Article{}, fmt.Errorf("décision sur l'article : %w", err)
	}
	return decided, nil
}

func decideInTransaction(ctx context.Context, tx pgx.Tx, decision annonces.Decision) (annonces.Article, error) {
	row := tx.QueryRow(ctx, decideQuery,
		uuid.UUID(decision.Article), int(decision.Revision), string(annonces.StatusToVerify),
		string(decision.Verdict.Outcome()), reasonColumn(decision.Reason),
	)
	return scanArticle(row)
}

func recordDecision(ctx context.Context, tx pgx.Tx, decision annonces.Decision) error {
	_, err := tx.Exec(ctx, recordDecisionQuery,
		uuid.UUID(decision.Article), int(decision.Revision), string(decision.Verdict),
		reasonColumn(decision.Reason), uuid.UUID(decision.Admin),
	)
	return err
}

func (s *Store) explainDecisionConflict(ctx context.Context, decision annonces.Decision) error {
	var (
		status   string
		revision int
	)
	err := s.pool.QueryRow(ctx, articleStateQuery, uuid.UUID(decision.Article)).Scan(&status, &revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return annonces.ErrArticleNotFound
	}
	if err != nil {
		return fmt.Errorf("lecture de l'état de l'article : %w", err)
	}
	return annonces.DecisionConflict(annonces.Status(status), annonces.Revision(revision), decision.Revision)
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
