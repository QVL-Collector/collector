package api

import (
	"errors"
	"net/http"
	"net/url"

	"collector/libs/platform/httpjson"
	"collector/libs/platform/problem"
	"collector/services/place-de-marche/internal/annonces"
)

var unknownCategoryFilter = []problem.FieldError{{Field: categoryParameter, Code: string(annonces.CodeUnknownCategory)}}

type catalogueEntryResponse struct {
	ID       string           `json:"id"`
	Title    string           `json:"titre"`
	Price    moneyResponse    `json:"prix"`
	Category categoryResponse `json:"categorie"`
}

type publishedArticleResponse struct {
	catalogueEntryResponse
	Description string         `json:"description"`
	ShippingFee *moneyResponse `json:"fraisDePort"`
}

type cataloguePageResponse struct {
	Entries []catalogueEntryResponse `json:"elements"`
	Page    int                      `json:"page"`
	Size    int                      `json:"taille"`
	Total   int                      `json:"total"`
}

func (h handlers) listCatalogue(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter, filterErrors := catalogueFilterOf(query)
	request, pageErrors := pageRequestOf(query)
	if fieldErrors := append(filterErrors, pageErrors...); len(fieldErrors) > 0 {
		problem.Write(w, problem.Validation(fieldErrors))
		return
	}
	page, err := h.service.Catalogue(r.Context(), filter, request)
	if errors.Is(err, annonces.ErrUnknownCategory) {
		problem.Write(w, problem.Validation(unknownCategoryFilter))
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, toCataloguePageResponse(page, request))
}

func (h handlers) publishedArticle(w http.ResponseWriter, r *http.Request) {
	id, err := annonces.ParseArticleID(r.PathValue("id"))
	if err != nil {
		problem.Write(w, problem.NotFound())
		return
	}
	article, err := h.service.PublishedArticle(r.Context(), id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpjson.Write(w, http.StatusOK, toPublishedArticleResponse(article))
}

func catalogueFilterOf(query url.Values) (annonces.CatalogueFilter, []problem.FieldError) {
	raw, present, fieldErrors := singleParameter(query, categoryParameter)
	if !present || len(fieldErrors) > 0 {
		return annonces.CatalogueFilter{}, fieldErrors
	}
	category := annonces.CategoryID(raw)
	return annonces.CatalogueFilter{Category: &category}, nil
}

func toCatalogueEntryResponse(entry annonces.CatalogueEntry) catalogueEntryResponse {
	return catalogueEntryResponse{
		ID:       entry.ID.String(),
		Title:    entry.Title,
		Price:    moneyResponseOf(entry.Price),
		Category: categoryResponseOf(entry.Category),
	}
}

func toPublishedArticleResponse(article annonces.PublishedArticle) publishedArticleResponse {
	return publishedArticleResponse{
		catalogueEntryResponse: toCatalogueEntryResponse(article.CatalogueEntry),
		Description:            article.Description,
		ShippingFee:            toMoneyResponse(article.ShippingFee),
	}
}

func toCataloguePageResponse(page annonces.CataloguePage, request annonces.PageRequest) cataloguePageResponse {
	entries := make([]catalogueEntryResponse, 0, len(page.Entries))
	for _, entry := range page.Entries {
		entries = append(entries, toCatalogueEntryResponse(entry))
	}
	return cataloguePageResponse{Entries: entries, Page: request.Number, Size: request.Size, Total: page.Total}
}
