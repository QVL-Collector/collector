package canal

import (
	"collector/libs/platform/message"
	"collector/services/controle/internal/regles"
)

const (
	RequestQueue    = "controle.demandes"
	resultExchange  = "controle.resultats"
	routingKeyV1    = "v1"
	contractVersion = 1
)

var (
	requestContract = message.Contract{Type: "controle.demande", Version: contractVersion}
	resultContract  = message.Contract{Type: "controle.resultat", Version: contractVersion}
)

type requestData struct {
	ArticleID string      `json:"article_id"`
	Revision  int         `json:"revision"`
	Listing   listingData `json:"listing"`
}

type listingData struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	CategoryCode *string `json:"category_code"`
	PriceCents   *int64  `json:"price_cents"`
}

func (l listingData) listing() regles.Listing {
	return regles.Listing{
		Title:        l.Title,
		Description:  l.Description,
		CategoryCode: (*regles.CategoryCode)(l.CategoryCode),
		PriceCents:   l.PriceCents,
	}
}

type resultData struct {
	ArticleID    string  `json:"article_id"`
	Revision     int     `json:"revision"`
	RulesVersion int     `json:"rules_version"`
	Issue        string  `json:"issue"`
	Reason       *string `json:"reason,omitempty"`
}

func toResultData(request requestData, verdict regles.Verdict) resultData {
	result := resultData{
		ArticleID:    request.ArticleID,
		Revision:     request.Revision,
		RulesVersion: regles.RulesVersion,
		Issue:        string(verdict.Issue),
	}
	if verdict.HasReason() {
		reason := string(verdict.Reason)
		result.Reason = &reason
	}
	return result
}
