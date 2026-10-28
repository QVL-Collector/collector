package stockage

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"collector/services/place-de-marche/internal/annonces"
)

const catalogueEntryColumns = `a.id, a.titre, c.id, c.libelle, a.prix_centimes`

const publishedArticles = `
FROM article AS a JOIN categorie AS c ON c.id = a.categorie_id
WHERE a.etat = 'publie' AND a.prix_centimes IS NOT NULL`

const publicationOrder = `
ORDER BY a.modifie_le DESC, a.id DESC`

const (
	countPublishedQuery           = `SELECT count(*)` + publishedArticles
	countPublishedInCategoryQuery = countPublishedQuery + ` AND a.categorie_id = $1`
	listPublishedQuery            = `SELECT ` + catalogueEntryColumns + publishedArticles + publicationOrder + `
LIMIT $1 OFFSET $2`
	listPublishedInCategoryQuery = `SELECT ` + catalogueEntryColumns + publishedArticles + ` AND a.categorie_id = $3` + publicationOrder + `
LIMIT $1 OFFSET $2`
	findPublishedQuery = `SELECT ` + catalogueEntryColumns + `, a.description, a.frais_port_centimes` + publishedArticles + `
AND a.id = $1`
)

var readOnlySnapshot = pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly}

type catalogueQueries struct {
	count      string
	list       string
	filterArgs []any
}

func catalogueQueriesFor(filter annonces.CatalogueFilter) catalogueQueries {
	if filter.Category == nil {
		return catalogueQueries{count: countPublishedQuery, list: listPublishedQuery}
	}
	return catalogueQueries{
		count:      countPublishedInCategoryQuery,
		list:       listPublishedInCategoryQuery,
		filterArgs: []any{string(*filter.Category)},
	}
}

func (s *Store) ListPublished(ctx context.Context, filter annonces.CatalogueFilter, page annonces.PageRequest) (annonces.CataloguePage, error) {
	queries := catalogueQueriesFor(filter)
	var catalogue annonces.CataloguePage
	err := pgx.BeginTxFunc(ctx, s.pool, readOnlySnapshot, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, queries.count, queries.filterArgs...).Scan(&catalogue.Total); err != nil {
			return fmt.Errorf("décompte du catalogue : %w", err)
		}
		rows, err := tx.Query(ctx, queries.list, append([]any{page.Size, page.Offset()}, queries.filterArgs...)...)
		if err != nil {
			return fmt.Errorf("liste du catalogue : %w", err)
		}
		catalogue.Entries, err = pgx.CollectRows(rows, func(row pgx.CollectableRow) (annonces.CatalogueEntry, error) {
			var r catalogueEntryRow
			err := row.Scan(r.targets()...)
			return r.entry(), err
		})
		return err
	})
	if err != nil {
		return annonces.CataloguePage{}, fmt.Errorf("lecture du catalogue : %w", err)
	}
	return catalogue, nil
}

func (s *Store) FindPublished(ctx context.Context, id annonces.ArticleID) (annonces.PublishedArticle, error) {
	var r publishedArticleRow
	err := pgx.BeginTxFunc(ctx, s.pool, readOnlySnapshot, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, findPublishedQuery, uuid.UUID(id)).Scan(r.targets()...)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return annonces.PublishedArticle{}, annonces.ErrArticleNotFound
	}
	if err != nil {
		return annonces.PublishedArticle{}, fmt.Errorf("lecture de l'article publié : %w", err)
	}
	return r.article(), nil
}

type catalogueEntryRow struct {
	id            uuid.UUID
	title         string
	categoryID    string
	categoryLabel string
	priceCents    int64
}

func (r *catalogueEntryRow) targets() []any {
	return []any{&r.id, &r.title, &r.categoryID, &r.categoryLabel, &r.priceCents}
}

func (r catalogueEntryRow) entry() annonces.CatalogueEntry {
	return annonces.CatalogueEntry{
		ID:       annonces.ArticleID(r.id),
		Title:    r.title,
		Category: annonces.Category{ID: annonces.CategoryID(r.categoryID), Label: r.categoryLabel},
		Price:    annonces.Money{Cents: r.priceCents, Currency: annonces.Euro},
	}
}

type publishedArticleRow struct {
	catalogueEntryRow
	description      string
	shippingFeeCents *int64
}

func (r *publishedArticleRow) targets() []any {
	return append(r.catalogueEntryRow.targets(), &r.description, &r.shippingFeeCents)
}

func (r publishedArticleRow) article() annonces.PublishedArticle {
	return annonces.PublishedArticle{
		CatalogueEntry: r.entry(),
		Description:    r.description,
		ShippingFee:    euros(r.shippingFeeCents),
	}
}
