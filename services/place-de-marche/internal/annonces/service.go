package annonces

import (
	"context"
	"fmt"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 50
	MaxPageNumber   = 10_000
)

type PageRequest struct {
	Number int
	Size   int
}

func (p PageRequest) Offset() int {
	return (p.Number - 1) * p.Size
}

type ArticlePage struct {
	Articles []Article
	Total    int
}

type Articles interface {
	CreateDraft(ctx context.Context, seller SellerID, content Content) (Article, error)
	UpdateDraft(ctx context.Context, seller SellerID, id ArticleID, expected Revision, content Content) (Article, error)
	FindOwned(ctx context.Context, seller SellerID, id ArticleID) (Article, error)
	ListOwned(ctx context.Context, seller SellerID, page PageRequest) (ArticlePage, error)
	Submit(ctx context.Context, seller SellerID, id ArticleID, expected Revision) (Article, error)
	ApplyReview(ctx context.Context, outcome ReviewOutcome) (bool, error)
	ListPublished(ctx context.Context, filter CatalogueFilter, page PageRequest) (CataloguePage, error)
	FindPublished(ctx context.Context, id ArticleID) (PublishedArticle, error)
}

type Categories interface {
	ListCategories(ctx context.Context) ([]Category, error)
	CategoryExists(ctx context.Context, id CategoryID) (bool, error)
}

type Service struct {
	articles   Articles
	categories Categories
}

func NewService(articles Articles, categories Categories) *Service {
	return &Service{articles: articles, categories: categories}
}

func (s *Service) CreateDraft(ctx context.Context, seller SellerID, content Content) (Article, error) {
	if err := s.validate(ctx, content); err != nil {
		return Article{}, err
	}
	return s.articles.CreateDraft(ctx, seller, content)
}

func (s *Service) UpdateDraft(ctx context.Context, seller SellerID, id ArticleID, expected Revision, content Content) (Article, error) {
	if err := s.validate(ctx, content); err != nil {
		return Article{}, err
	}
	return s.articles.UpdateDraft(ctx, seller, id, expected, content)
}

func (s *Service) OwnArticle(ctx context.Context, seller SellerID, id ArticleID) (Article, error) {
	return s.articles.FindOwned(ctx, seller, id)
}

func (s *Service) OwnArticles(ctx context.Context, seller SellerID, page PageRequest) (ArticlePage, error) {
	return s.articles.ListOwned(ctx, seller, page)
}

func (s *Service) Submit(ctx context.Context, seller SellerID, id ArticleID, expected Revision) (Article, error) {
	return s.articles.Submit(ctx, seller, id, expected)
}

func (s *Service) ApplyReview(ctx context.Context, outcome ReviewOutcome) (bool, error) {
	return s.articles.ApplyReview(ctx, outcome)
}

func (s *Service) Categories(ctx context.Context) ([]Category, error) {
	return s.categories.ListCategories(ctx)
}

func (s *Service) validate(ctx context.Context, content Content) error {
	violations := content.violations()
	if content.CategoryID != nil && !hasViolationOn(violations, FieldCategory) {
		exists, err := s.categories.CategoryExists(ctx, *content.CategoryID)
		if err != nil {
			return fmt.Errorf("vérification de la catégorie : %w", err)
		}
		if !exists {
			violations = append(violations, Violation{Field: FieldCategory, Code: CodeUnknownCategory})
		}
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}
	return nil
}
