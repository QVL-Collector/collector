package annonces

import (
	"context"
	"errors"
	"fmt"
)

var ErrUnknownCategory = errors.New("catégorie inconnue")

type CatalogueFilter struct {
	Category *CategoryID
}

type CatalogueEntry struct {
	ID       ArticleID
	Title    string
	Category Category
	Price    Money
}

type PublishedArticle struct {
	CatalogueEntry
	Description string
	ShippingFee *Money
}

type CataloguePage struct {
	Entries []CatalogueEntry
	Total   int
}

func (s *Service) Catalogue(ctx context.Context, filter CatalogueFilter, page PageRequest) (CataloguePage, error) {
	if err := s.checkFilter(ctx, filter); err != nil {
		return CataloguePage{}, err
	}
	return s.articles.ListPublished(ctx, filter, page)
}

func (s *Service) PublishedArticle(ctx context.Context, id ArticleID) (PublishedArticle, error) {
	return s.articles.FindPublished(ctx, id)
}

func (s *Service) checkFilter(ctx context.Context, filter CatalogueFilter) error {
	if filter.Category == nil {
		return nil
	}
	if !filter.Category.wellFormed() {
		return ErrUnknownCategory
	}
	exists, err := s.categories.CategoryExists(ctx, *filter.Category)
	if err != nil {
		return fmt.Errorf("vérification de la catégorie filtrée : %w", err)
	}
	if !exists {
		return ErrUnknownCategory
	}
	return nil
}
