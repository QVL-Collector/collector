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

var (
	etatsModifiablesSansControle    = []annonces.Status{annonces.StatusDraft, annonces.StatusRefused}
	etatsRecontrolesALaModification = []annonces.Status{annonces.StatusPublished, annonces.StatusUnderReview, annonces.StatusToVerify}
)

type stockageEnMemoire struct {
	mu         sync.Mutex
	articles   map[annonces.ArticleID]annonces.Article
	categories []annonces.Category
	instant    time.Time
	demandes   []annonces.ReviewRequest
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
	recontrole := slices.Contains(etatsRecontrolesALaModification, article.Status)
	if !recontrole && !slices.Contains(etatsModifiablesSansControle, article.Status) {
		return annonces.Article{}, annonces.ErrArticleNotEditable
	}
	if article.Revision != attendue {
		return annonces.Article{}, annonces.ErrStaleRevision
	}
	s.appliquer(&article, contenu)
	article.Revision++
	if recontrole {
		article.Status = annonces.StatusUnderReview
		article.Reason = nil
		s.demander(article)
	}
	s.articles[id] = article
	return article, nil
}

func (s *stockageEnMemoire) demander(article annonces.Article) {
	s.demandes = append(s.demandes, annonces.ReviewRequest{
		Article:     article,
		Seller:      annonces.SellerStatus{MemberSince: time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)},
		SubmittedAt: article.UpdatedAt,
	})
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

func (s *stockageEnMemoire) Submit(_ context.Context, vendeur annonces.SellerID, id annonces.ArticleID, attendue annonces.Revision) (annonces.Article, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	article, ok := s.articles[id]
	if !ok || article.Seller != vendeur {
		return annonces.Article{}, annonces.ErrArticleNotFound
	}
	if !article.Status.Submittable() {
		return annonces.Article{}, annonces.ErrArticleNotSubmittable
	}
	if article.Revision != attendue {
		return annonces.Article{}, annonces.ErrStaleRevision
	}
	article.Status = annonces.StatusUnderReview
	article.Revision++
	article.Reason = nil
	article.UpdatedAt = s.maintenant()
	s.articles[id] = article
	s.demander(article)
	return article, nil
}

func (s *stockageEnMemoire) ApplyReview(_ context.Context, issue annonces.ReviewOutcome) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	article, ok := s.articles[issue.Article]
	if !ok || article.Status != annonces.StatusUnderReview || article.Revision != issue.Revision {
		return false, nil
	}
	article.Status = issue.Status
	article.Reason = issue.Reason
	article.UpdatedAt = s.maintenant()
	s.articles[issue.Article] = article
	return true, nil
}

func (s *stockageEnMemoire) publies(filtre annonces.CatalogueFilter) []annonces.Article {
	var publies []annonces.Article
	for _, a := range s.articles {
		if a.Status != annonces.StatusPublished || a.Category == nil {
			continue
		}
		if filtre.Category == nil || a.Category.ID == *filtre.Category {
			publies = append(publies, a)
		}
	}
	slices.SortFunc(publies, func(a, b annonces.Article) int { return b.UpdatedAt.Compare(a.UpdatedAt) })
	return publies
}

func entreeDuCatalogue(a annonces.Article) annonces.CatalogueEntry {
	entree := annonces.CatalogueEntry{ID: a.ID, Title: a.Title, Category: *a.Category}
	if a.Price != nil {
		entree.Price = *a.Price
	}
	return entree
}

func (s *stockageEnMemoire) ListPublished(_ context.Context, filtre annonces.CatalogueFilter, page annonces.PageRequest) (annonces.CataloguePage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	publies := s.publies(filtre)
	debut := min(page.Offset(), len(publies))
	fin := min(debut+page.Size, len(publies))
	entrees := make([]annonces.CatalogueEntry, 0, fin-debut)
	for _, a := range publies[debut:fin] {
		entrees = append(entrees, entreeDuCatalogue(a))
	}
	return annonces.CataloguePage{Entries: entrees, Total: len(publies)}, nil
}

func (s *stockageEnMemoire) FindPublished(_ context.Context, id annonces.ArticleID) (annonces.PublishedArticle, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	article, ok := s.articles[id]
	if !ok || article.Status != annonces.StatusPublished || article.Category == nil {
		return annonces.PublishedArticle{}, annonces.ErrArticleNotFound
	}
	return annonces.PublishedArticle{
		CatalogueEntry: entreeDuCatalogue(article),
		Description:    article.Description,
		ShippingFee:    article.ShippingFee,
	}, nil
}

func (s *stockageEnMemoire) retirerDemandes() []annonces.ReviewRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	demandes := s.demandes
	s.demandes = nil
	return demandes
}
