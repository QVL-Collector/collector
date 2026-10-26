package api

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	"collector/libs/platform/httpjson"
	"collector/libs/platform/httpserver"
	"collector/libs/platform/problem"
	"collector/libs/platform/token"
	"collector/services/place-de-marche/internal/annonces"
)

const (
	basePath          = "/marche/v1"
	sellerArticles    = basePath + "/vendeur/articles"
	sellerRole        = "vendeur"
	maxBodyBytes      = 32 << 10
	pageParameter     = "page"
	pageSizeParameter = "taille"
)

var staleRevision = problem.Typed("urn:collector:probleme:revision-perimee", "Révision périmée",
	http.StatusConflict, "L'article a été modifié depuis sa dernière lecture.")

var notEditable = problem.Typed("urn:collector:probleme:article-non-modifiable", "Article non modifiable",
	http.StatusConflict, "L'article ne peut pas être modifié dans son état actuel.")

var notSubmittable = problem.Typed("urn:collector:probleme:article-non-soumettable", "Article non soumettable",
	http.StatusConflict, "L'article ne peut pas être soumis au contrôle dans son état actuel.")

type ArticleService interface {
	CreateDraft(ctx context.Context, seller annonces.SellerID, content annonces.Content) (annonces.Article, error)
	UpdateDraft(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, expected annonces.Revision, content annonces.Content) (annonces.Article, error)
	Submit(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, expected annonces.Revision) (annonces.Article, error)
	OwnArticle(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID) (annonces.Article, error)
	OwnArticles(ctx context.Context, seller annonces.SellerID, page annonces.PageRequest) (annonces.ArticlePage, error)
	Categories(ctx context.Context) ([]annonces.Category, error)
}

type Authenticator interface {
	Authenticate(logger *slog.Logger, next http.Handler) http.Handler
}

type handlers struct {
	service ArticleService
	logger  *slog.Logger
}

func Register(mux *http.ServeMux, service ArticleService, authenticator Authenticator, logger *slog.Logger) {
	h := handlers{service: service, logger: logger}
	sellerOnly := func(handler http.HandlerFunc) http.Handler {
		return authenticator.Authenticate(logger, token.RequireRole(sellerRole, handler))
	}
	mux.HandleFunc("GET "+basePath+"/categories", h.listCategories)
	mux.Handle("POST "+sellerArticles, sellerOnly(h.createDraft))
	mux.Handle("GET "+sellerArticles, sellerOnly(h.listOwnArticles))
	mux.Handle("GET "+sellerArticles+"/{id}", sellerOnly(h.ownArticle))
	mux.Handle("PUT "+sellerArticles+"/{id}", sellerOnly(h.updateDraft))
	mux.Handle("POST "+sellerArticles+"/{id}/soumission", sellerOnly(h.submit))
	mux.HandleFunc("/", httpserver.NotFound)
}

func (h handlers) listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.Categories(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, toCategoryListResponse(categories))
}

func (h handlers) createDraft(w http.ResponseWriter, r *http.Request) {
	seller, authenticated := sellerOf(w, r)
	if !authenticated {
		return
	}
	var payload draftPayload
	if !h.decode(w, r, &payload) {
		return
	}
	content, fieldErrors := payload.content()
	if len(fieldErrors) > 0 {
		problem.Write(w, problem.Validation(fieldErrors))
		return
	}
	article, err := h.service.CreateDraft(r.Context(), seller, content)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Location", sellerArticles+"/"+article.ID.String())
	httpjson.Write(w, http.StatusCreated, toArticleResponse(article))
}

func (h handlers) updateDraft(w http.ResponseWriter, r *http.Request) {
	seller, id, found := ownedArticleOf(w, r)
	if !found {
		return
	}
	var payload draftUpdatePayload
	if !h.decode(w, r, &payload) {
		return
	}
	expected, revisionErrors := payload.revision()
	content, contentErrors := payload.content()
	if fieldErrors := append(revisionErrors, contentErrors...); len(fieldErrors) > 0 {
		problem.Write(w, problem.Validation(fieldErrors))
		return
	}
	article, err := h.service.UpdateDraft(r.Context(), seller, id, expected, content)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, toArticleResponse(article))
}

func (h handlers) submit(w http.ResponseWriter, r *http.Request) {
	seller, id, found := ownedArticleOf(w, r)
	if !found {
		return
	}
	var payload submissionPayload
	if !h.decode(w, r, &payload) {
		return
	}
	expected, fieldErrors := revisionOf(payload.Revision)
	if len(fieldErrors) > 0 {
		problem.Write(w, problem.Validation(fieldErrors))
		return
	}
	article, err := h.service.Submit(r.Context(), seller, id, expected)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusAccepted, toArticleResponse(article))
}

func (h handlers) ownArticle(w http.ResponseWriter, r *http.Request) {
	seller, id, found := ownedArticleOf(w, r)
	if !found {
		return
	}
	article, err := h.service.OwnArticle(r.Context(), seller, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, toArticleResponse(article))
}

func (h handlers) listOwnArticles(w http.ResponseWriter, r *http.Request) {
	seller, authenticated := sellerOf(w, r)
	if !authenticated {
		return
	}
	request, fieldErrors := pageRequestOf(r.URL.Query())
	if len(fieldErrors) > 0 {
		problem.Write(w, problem.Validation(fieldErrors))
		return
	}
	page, err := h.service.OwnArticles(r.Context(), seller, request)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, toArticlePageResponse(page, request))
}

func sellerOf(w http.ResponseWriter, r *http.Request) (annonces.SellerID, bool) {
	claims, found := token.ClaimsFrom(r.Context())
	if !found {
		token.Unauthorized(w)
		return annonces.SellerID{}, false
	}
	seller, err := annonces.ParseSellerID(claims.Subject)
	if err != nil {
		token.Unauthorized(w)
		return annonces.SellerID{}, false
	}
	return seller, true
}

func ownedArticleOf(w http.ResponseWriter, r *http.Request) (annonces.SellerID, annonces.ArticleID, bool) {
	seller, authenticated := sellerOf(w, r)
	if !authenticated {
		return annonces.SellerID{}, annonces.ArticleID{}, false
	}
	id, err := annonces.ParseArticleID(r.PathValue("id"))
	if err != nil {
		problem.Write(w, problem.NotFound())
		return annonces.SellerID{}, annonces.ArticleID{}, false
	}
	return seller, id, true
}

func (h handlers) decode(w http.ResponseWriter, r *http.Request, target any) bool {
	err := httpjson.Decode(w, r, maxBodyBytes, target)
	if err == nil {
		return true
	}
	var rejection *httpjson.RejectionError
	if errors.As(err, &rejection) {
		problem.Write(w, rejection.Problem())
		return false
	}
	h.writeError(w, r, err)
	return false
}

func pageRequestOf(query url.Values) (annonces.PageRequest, []problem.FieldError) {
	number, numberErrors := boundedParameter(query, pageParameter, 1, annonces.MaxPageNumber)
	size, sizeErrors := boundedParameter(query, pageSizeParameter, annonces.DefaultPageSize, annonces.MaxPageSize)
	return annonces.PageRequest{Number: number, Size: size}, append(numberErrors, sizeErrors...)
}

func boundedParameter(query url.Values, name string, fallback, maximum int) (int, []problem.FieldError) {
	values, present := query[name]
	if !present {
		return fallback, nil
	}
	if len(values) != 1 {
		return 0, []problem.FieldError{{Field: name, Code: codeInvalid}}
	}
	value, err := strconv.Atoi(values[0])
	if err != nil {
		return 0, []problem.FieldError{{Field: name, Code: codeInvalid}}
	}
	if value < 1 || value > maximum {
		return 0, []problem.FieldError{{Field: name, Code: codeOutOfBounds}}
	}
	return value, nil
}

func (h handlers) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *annonces.ValidationError
	switch {
	case errors.As(err, &invalid):
		problem.Write(w, problem.Validation(violationErrors(invalid.Violations)))
	case errors.Is(err, annonces.ErrArticleNotFound):
		problem.Write(w, problem.NotFound())
	case errors.Is(err, annonces.ErrStaleRevision):
		problem.Write(w, staleRevision)
	case errors.Is(err, annonces.ErrArticleNotEditable):
		problem.Write(w, notEditable)
	case errors.Is(err, annonces.ErrArticleNotSubmittable):
		problem.Write(w, notSubmittable)
	default:
		h.logger.ErrorContext(r.Context(), "erreur interne", slog.String("event", "annonces.erreur_interne"), slog.Any("erreur", err))
		problem.Write(w, problem.Internal())
	}
}
