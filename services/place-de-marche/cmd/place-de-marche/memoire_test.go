package main

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"

	"collector/services/place-de-marche/internal/annonces"
)

var jeuDeCategoriesFictif = []annonces.Category{
	{ID: "cartes-postales", Label: "Cartes postales"},
	{ID: "figurines", Label: "Figurines"},
	{ID: "timbres", Label: "Timbres"},
}

type stockageEnMemoire struct {
	mu         sync.Mutex
	articles   map[annonces.ArticleID]annonces.Article
	categories []annonces.Category
	instant    time.Time
}

func nouveauStockageEnMemoire(categories []annonces.Category) *stockageEnMemoire {
	return &stockageEnMemoire{
		articles:   map[annonces.ArticleID]annonces.Article{},
		categories: slices.Clone(categories),
		instant:    time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC),
	}
}

func (s *stockageEnMemoire) maintenant() time.Time {
	s.instant = s.instant.Add(time.Second)
	return s.instant
}

func (s *stockageEnMemoire) categorie(id *annonces.CategoryID) *annonces.Category {
	if id == nil {
		return nil
	}
	for _, c := range s.categories {
		if c.ID == *id {
			trouvee := c
			return &trouvee
		}
	}
	return nil
}

func (s *stockageEnMemoire) appliquer(article *annonces.Article, contenu annonces.Content) {
	article.Title = contenu.Title
	article.Description = contenu.Description
	article.Category = s.categorie(contenu.CategoryID)
	article.Price = contenu.Price
	article.ShippingFee = contenu.ShippingFee
	article.UpdatedAt = s.maintenant()
}

func (s *stockageEnMemoire) CreateDraft(_ context.Context, vendeur annonces.SellerID, contenu annonces.Content) (annonces.Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	article := annonces.Article{
		ID:       annonces.ArticleID(uuid.New()),
		Seller:   vendeur,
		Status:   annonces.StatusDraft,
		Revision: annonces.FirstRevision,
	}
	s.appliquer(&article, contenu)
	article.CreatedAt = article.UpdatedAt
	s.articles[article.ID] = article
	return article, nil
}

func (s *stockageEnMemoire) UpdateDraft(_ context.Context, vendeur annonces.SellerID, id annonces.ArticleID, attendue annonces.Revision, contenu annonces.Content) (annonces.Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	article, ok := s.articles[id]
	if !ok || article.Seller != vendeur {
		return annonces.Article{}, annonces.ErrArticleNotFound
	}
	if !article.Status.Editable() {
		return annonces.Article{}, annonces.ErrArticleNotEditable
	}
	if article.Revision != attendue {
		return annonces.Article{}, annonces.ErrStaleRevision
	}
	s.appliquer(&article, contenu)
	article.Revision++
	s.articles[id] = article
	return article, nil
}

func (s *stockageEnMemoire) FindOwned(_ context.Context, vendeur annonces.SellerID, id annonces.ArticleID) (annonces.Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	article, ok := s.articles[id]
	if !ok || article.Seller != vendeur {
		return annonces.Article{}, annonces.ErrArticleNotFound
	}
	return article, nil
}

func (s *stockageEnMemoire) ListOwned(_ context.Context, vendeur annonces.SellerID, page annonces.PageRequest) (annonces.ArticlePage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var siens []annonces.Article
	for _, a := range s.articles {
		if a.Seller == vendeur {
			siens = append(siens, a)
		}
	}
	slices.SortFunc(siens, func(a, b annonces.Article) int { return b.UpdatedAt.Compare(a.UpdatedAt) })
	debut := min(page.Offset(), len(siens))
	fin := min(debut+page.Size, len(siens))
	return annonces.ArticlePage{Articles: siens[debut:fin], Total: len(siens)}, nil
}

func (s *stockageEnMemoire) ListCategories(_ context.Context) ([]annonces.Category, error) {
	return slices.Clone(s.categories), nil
}

func (s *stockageEnMemoire) CategoryExists(_ context.Context, id annonces.CategoryID) (bool, error) {
	return s.categorie(&id) != nil, nil
}
