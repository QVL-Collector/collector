package api

import (
	"net/http"
	"time"

	"collector/libs/platform/problem"
	"collector/services/place-de-marche/internal/annonces"
)

const (
	codeRequired    = "obligatoire"
	codeInvalid     = "invalide"
	codeOutOfBounds = "hors_limites"
	amountMember    = ".montantCentimes"
	currencyMember  = ".devise"
	priceField      = "prix"
	shippingField   = "fraisDePort"
	revisionField   = "revision"
	verdictField    = "decision"
	reasonField     = "motif"
)

var fieldNames = map[annonces.Field]string{
	annonces.FieldTitle:               "titre",
	annonces.FieldDescription:         "description",
	annonces.FieldCategory:            "categorieId",
	annonces.FieldPrice:               priceField + amountMember,
	annonces.FieldPriceCurrency:       priceField + currencyMember,
	annonces.FieldShippingFee:         shippingField + amountMember,
	annonces.FieldShippingFeeCurrency: shippingField + currencyMember,
}

type moneyPayload struct {
	AmountInCents *int64 `json:"montantCentimes"`
	Currency      string `json:"devise"`
}

func (m *moneyPayload) money(field string) (*annonces.Money, []problem.FieldError) {
	if m == nil {
		return nil, nil
	}
	if m.AmountInCents == nil || m.Currency == "" {
		return nil, m.missingMembers(field)
	}
	return &annonces.Money{Cents: *m.AmountInCents, Currency: annonces.Currency(m.Currency)}, nil
}

func (m *moneyPayload) missingMembers(field string) []problem.FieldError {
	var missing []problem.FieldError
	if m.AmountInCents == nil {
		missing = append(missing, problem.FieldError{Field: field + amountMember, Code: codeRequired})
	}
	if m.Currency == "" {
		missing = append(missing, problem.FieldError{Field: field + currencyMember, Code: codeRequired})
	}
	return missing
}

type draftPayload struct {
	Title       string        `json:"titre"`
	Description string        `json:"description"`
	CategoryID  *string       `json:"categorieId"`
	Price       *moneyPayload `json:"prix"`
	ShippingFee *moneyPayload `json:"fraisDePort"`
}

func (p draftPayload) content() (annonces.Content, []problem.FieldError) {
	price, priceErrors := p.Price.money(priceField)
	shippingFee, shippingErrors := p.ShippingFee.money(shippingField)
	content := annonces.Content{
		Title:       p.Title,
		Description: p.Description,
		CategoryID:  categoryID(p.CategoryID),
		Price:       price,
		ShippingFee: shippingFee,
	}
	return content, append(priceErrors, shippingErrors...)
}

func categoryID(raw *string) *annonces.CategoryID {
	if raw == nil {
		return nil
	}
	id := annonces.CategoryID(*raw)
	return &id
}

type draftUpdatePayload struct {
	Revision *int `json:"revision"`
	draftPayload
}

func (p draftUpdatePayload) revision() (annonces.Revision, []problem.FieldError) {
	return revisionOf(p.Revision)
}

type submissionPayload struct {
	Revision *int `json:"revision"`
}

func revisionOf(raw *int) (annonces.Revision, []problem.FieldError) {
	if raw == nil {
		return 0, []problem.FieldError{{Field: revisionField, Code: codeRequired}}
	}
	if *raw < int(annonces.FirstRevision) {
		return 0, []problem.FieldError{{Field: revisionField, Code: codeInvalid}}
	}
	return annonces.Revision(*raw), nil
}

type decisionPayload struct {
	Revision *int    `json:"revision"`
	Verdict  *string `json:"decision"`
	Reason   *string `json:"motif"`
}

func (p decisionPayload) decision(admin annonces.AdminID, id annonces.ArticleID) (annonces.Decision, problem.Problem, bool) {
	revision, revisionErrors := revisionOf(p.Revision)
	verdict, verdictErrors := verdictOf(p.Verdict)
	if fieldErrors := append(revisionErrors, verdictErrors...); len(fieldErrors) > 0 {
		return annonces.Decision{}, problem.Validation(fieldErrors), true
	}
	reason, reasonErrors := refusalReasonOf(verdict, p.Reason)
	if len(reasonErrors) > 0 {
		return annonces.Decision{}, invalidReason(reasonErrors), true
	}
	decision := annonces.Decision{Admin: admin, Article: id, Revision: revision, Verdict: verdict, Reason: reason}
	return decision, problem.Problem{}, false
}

func verdictOf(raw *string) (annonces.Verdict, []problem.FieldError) {
	if raw == nil {
		return "", []problem.FieldError{{Field: verdictField, Code: codeRequired}}
	}
	verdict, err := annonces.ParseVerdict(*raw)
	if err != nil {
		return "", []problem.FieldError{{Field: verdictField, Code: codeInvalid}}
	}
	return verdict, nil
}

func refusalReasonOf(verdict annonces.Verdict, raw *string) (*annonces.ReviewReason, []problem.FieldError) {
	if !verdict.RequiresReason() {
		if raw != nil {
			return nil, []problem.FieldError{{Field: reasonField, Code: codeInvalid}}
		}
		return nil, nil
	}
	if raw == nil {
		return nil, []problem.FieldError{{Field: reasonField, Code: codeRequired}}
	}
	reason, err := annonces.ParseRefusalReason(*raw)
	if err != nil {
		return nil, []problem.FieldError{{Field: reasonField, Code: codeInvalid}}
	}
	return &reason, nil
}

func invalidReason(fieldErrors []problem.FieldError) problem.Problem {
	invalid := problem.Validation(fieldErrors)
	invalid.Status = http.StatusUnprocessableEntity
	return invalid
}

func violationErrors(violations []annonces.Violation) []problem.FieldError {
	fieldErrors := make([]problem.FieldError, 0, len(violations))
	for _, violation := range violations {
		fieldErrors = append(fieldErrors, problem.FieldError{Field: fieldNames[violation.Field], Code: string(violation.Code)})
	}
	return fieldErrors
}

type moneyResponse struct {
	AmountInCents int64  `json:"montantCentimes"`
	Currency      string `json:"devise"`
}

type categoryResponse struct {
	ID    string `json:"id"`
	Label string `json:"libelle"`
}

type articleResponse struct {
	ID          string            `json:"id"`
	Status      string            `json:"etat"`
	Revision    int               `json:"revision"`
	Title       string            `json:"titre"`
	Description string            `json:"description"`
	Category    *categoryResponse `json:"categorie"`
	Price       *moneyResponse    `json:"prix"`
	ShippingFee *moneyResponse    `json:"fraisDePort"`
	Reason      *string           `json:"motif"`
	CreatedAt   time.Time         `json:"creeLe"`
	UpdatedAt   time.Time         `json:"modifieLe"`
}

type articlePageResponse struct {
	Articles []articleResponse `json:"elements"`
	Page     int               `json:"page"`
	Size     int               `json:"taille"`
	Total    int               `json:"total"`
}

type articleToVerifyResponse struct {
	ID          string            `json:"id"`
	Revision    int               `json:"revision"`
	Title       string            `json:"titre"`
	Description string            `json:"description"`
	Category    *categoryResponse `json:"categorie"`
	Price       *moneyResponse    `json:"prix"`
	ShippingFee *moneyResponse    `json:"fraisDePort"`
	Reason      string            `json:"motif"`
	SellerID    string            `json:"vendeurId"`
	UpdatedAt   time.Time         `json:"modifieLe"`
}

type articleToVerifyPageResponse struct {
	Articles []articleToVerifyResponse `json:"elements"`
	Page     int                       `json:"page"`
	Size     int                       `json:"taille"`
	Total    int                       `json:"total"`
}

type categoryListResponse struct {
	Categories []categoryResponse `json:"elements"`
}

func toArticleResponse(article annonces.Article) articleResponse {
	return articleResponse{
		ID:          article.ID.String(),
		Status:      string(article.Status),
		Revision:    int(article.Revision),
		Title:       article.Title,
		Description: article.Description,
		Category:    toCategoryResponse(article.Category),
		Price:       toMoneyResponse(article.Price),
		ShippingFee: toMoneyResponse(article.ShippingFee),
		Reason:      toReasonResponse(article.Reason),
		CreatedAt:   article.CreatedAt.UTC(),
		UpdatedAt:   article.UpdatedAt.UTC(),
	}
}

func toArticlePageResponse(page annonces.ArticlePage, request annonces.PageRequest) articlePageResponse {
	articles := make([]articleResponse, 0, len(page.Articles))
	for _, article := range page.Articles {
		articles = append(articles, toArticleResponse(article))
	}
	return articlePageResponse{Articles: articles, Page: request.Number, Size: request.Size, Total: page.Total}
}

func toArticleToVerifyResponse(article annonces.Article) articleToVerifyResponse {
	return articleToVerifyResponse{
		ID:          article.ID.String(),
		Revision:    int(article.Revision),
		Title:       article.Title,
		Description: article.Description,
		Category:    toCategoryResponse(article.Category),
		Price:       toMoneyResponse(article.Price),
		ShippingFee: toMoneyResponse(article.ShippingFee),
		Reason:      reasonText(article.Reason),
		SellerID:    article.Seller.String(),
		UpdatedAt:   article.UpdatedAt.UTC(),
	}
}

func toArticleToVerifyPageResponse(page annonces.ArticlePage, request annonces.PageRequest) articleToVerifyPageResponse {
	articles := make([]articleToVerifyResponse, 0, len(page.Articles))
	for _, article := range page.Articles {
		articles = append(articles, toArticleToVerifyResponse(article))
	}
	return articleToVerifyPageResponse{Articles: articles, Page: request.Number, Size: request.Size, Total: page.Total}
}

func reasonText(reason *annonces.ReviewReason) string {
	if reason == nil {
		return ""
	}
	return string(*reason)
}

func toCategoryListResponse(categories []annonces.Category) categoryListResponse {
	responses := make([]categoryResponse, 0, len(categories))
	for _, category := range categories {
		responses = append(responses, categoryResponseOf(category))
	}
	return categoryListResponse{Categories: responses}
}

func toCategoryResponse(category *annonces.Category) *categoryResponse {
	if category == nil {
		return nil
	}
	response := categoryResponseOf(*category)
	return &response
}

func toReasonResponse(reason *annonces.ReviewReason) *string {
	if reason == nil {
		return nil
	}
	value := string(*reason)
	return &value
}

func toMoneyResponse(money *annonces.Money) *moneyResponse {
	if money == nil {
		return nil
	}
	response := moneyResponseOf(*money)
	return &response
}

func moneyResponseOf(money annonces.Money) moneyResponse {
	return moneyResponse{AmountInCents: money.Cents, Currency: string(money.Currency)}
}

func categoryResponseOf(category annonces.Category) categoryResponse {
	return categoryResponse{ID: string(category.ID), Label: category.Label}
}
