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
	publishedArticles = basePath + "/articles"
	adminArticles     = basePath + "/admin/articles"
	categoryParameter = "categorie"
	sellerRole        = "vendeur"
	adminRole         = "admin"
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

var notToVerify = problem.Typed("urn:collector:probleme:article-non-a-verifier", "Article non à vérifier",
	http.StatusConflict, "L'article n'est plus à vérifier.")

type ArticleService interface {
	CreateDraft(ctx context.Context, seller annonces.SellerID, content annonces.Content) (annonces.Article, error)
	UpdateDraft(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, expected annonces.Revision, content annonces.Content) (annonces.Article, error)
	Submit(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID, expected annonces.Revision) (annonces.Article, error)
	OwnArticle(ctx context.Context, seller annonces.SellerID, id annonces.ArticleID) (annonces.Article, error)
	OwnArticles(ctx context.Context, seller annonces.SellerID, page annonces.PageRequest) (annonces.ArticlePage, error)
	Categories(ctx context.Context) ([]annonces.Category, error)
	Catalogue(ctx context.Context, filter annonces.CatalogueFilter, page annonces.PageRequest) (annonces.CataloguePage, error)
	PublishedArticle(ctx context.Context, id annonces.ArticleID) (annonces.PublishedArticle, error)
	ArticlesToVerify(ctx context.Context, page annonces.PageRequest) (annonces.ArticlePage, error)
	Decide(ctx context.Context, decision annonces.Decision) (annonces.Article, error)
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
	restrictedTo := func(role string) func(http.HandlerFunc) http.Handler {
		return func(handler http.HandlerFunc) http.Handler {
			return authenticator.Authenticate(logger, token.RequireRole(role, handler))
		}
	}
	sellerOnly := restrictedTo(sellerRole)
	adminOnly := restrictedTo(adminRole)
	mux.HandleFunc("GET "+basePath+"/categories", h.listCategories)
	mux.HandleFunc("GET "+publishedArticles, h.listCatalogue)
	mux.HandleFunc("GET "+publishedArticles+"/{id}", h.publishedArticle)
	mux.Handle("POST "+sellerArticles, sellerOnly(h.createDraft))
	mux.Handle("GET "+sellerArticles, sellerOnly(h.listOwnArticles))
	mux.Handle("GET "+sellerArticles+"/{id}", sellerOnly(h.ownArticle))
	mux.Handle("PUT "+sellerArticles+"/{id}", sellerOnly(h.updateDraft))
	mux.Handle("POST "+sellerArticles+"/{id}/soumission", sellerOnly(h.submit))
	mux.Handle("GET "+adminArticles, adminOnly(h.listArticlesToVerify))
	mux.Handle("POST "+adminArticles+"/{id}/decision", adminOnly(h.decide))
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

func (h handlers) listArticlesToVerify(w http.ResponseWriter, r *http.Request) {
	request, fieldErrors := pageRequestOf(r.URL.Query())
	if len(fieldErrors) > 0 {
		problem.Write(w, problem.Validation(fieldErrors))
		return
	}
	page, err := h.service.ArticlesToVerify(r.Context(), request)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, toArticleToVerifyPageResponse(page, request))
}

func (h handlers) decide(w http.ResponseWriter, r *http.Request) {
	admin, authenticated := subjectOf(w, r, annonces.ParseAdminID)
	if !authenticated {
		return
	}
	id, err := annonces.ParseArticleID(r.PathValue("id"))
	if err != nil {
		problem.Write(w, problem.NotFound())
		return
	}
	var payload decisionPayload
	if !h.decode(w, r, &payload) {
		return
	}
	decision, rejection, rejected := payload.decision(admin, id)
	if rejected {
		problem.Write(w, rejection)
		return
	}
	article, err := h.service.Decide(r.Context(), decision)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.logDecision(r.Context(), decision)
	httpjson.Write(w, http.StatusOK, toArticleResponse(article))
}

func (h handlers) logDecision(ctx context.Context, decision annonces.Decision) {
	attributes := []any{
		slog.String("event", "annonces.decision_admin"),
		slog.String("admin_id", decision.Admin.String()),
		slog.String("article_id", decision.Article.String()),
		slog.Int("revision", int(decision.Revision)),
		slog.String("decision", string(decision.Verdict)),
	}
	if decision.Reason != nil {
		attributes = append(attributes, slog.String("motif", string(*decision.Reason)))
	}
	h.logger.InfoContext(ctx, "décision d'administration appliquée", attributes...)
}

func sellerOf(w http.ResponseWriter, r *http.Request) (annonces.SellerID, bool) {
	return subjectOf(w, r, annonces.ParseSellerID)
}

func subjectOf[ID any](w http.ResponseWriter, r *http.Request, parse func(string) (ID, error)) (ID, bool) {
	var none ID
	claims, found := token.ClaimsFrom(r.Context())
	if !found {
		token.Unauthorized(w)
		return none, false
	}
	id, err := parse(claims.Subject)
	if err != nil {
		token.Unauthorized(w)
		return none, false
	}
	return id, true
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
	raw, present, fieldErrors := singleParameter(query, name)
	if !present || len(fieldErrors) > 0 {
		return fallback, fieldErrors
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, []problem.FieldError{{Field: name, Code: codeInvalid}}
	}
	if value < 1 || value > maximum {
		return 0, []problem.FieldError{{Field: name, Code: codeOutOfBounds}}
	}
	return value, nil
}

func singleParameter(query url.Values, name string) (string, bool, []problem.FieldError) {
	values, present := query[name]
	if !present {
		return "", false, nil
	}
	if len(values) != 1 || values[0] == "" {
		return "", true, []problem.FieldError{{Field: name, Code: codeInvalid}}
	}
	return values[0], true, nil
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
	case errors.Is(err, annonces.ErrArticleNotToVerify):
		problem.Write(w, notToVerify)
	default:
		h.logger.ErrorContext(r.Context(), "erreur interne", slog.String("event", "annonces.erreur_interne"), slog.Any("erreur", err))
		problem.Write(w, problem.Internal())
	}
}
