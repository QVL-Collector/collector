package canal

import (
	"encoding/json"
	"fmt"
	"time"

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
	ArticleID   string      `json:"article_id"`
	Revision    int         `json:"revision"`
	SubmittedAt time.Time   `json:"submitted_at"`
	Seller      sellerData  `json:"seller"`
	Listing     listingData `json:"listing"`
}

func (r requestData) submission() regles.Submission {
	return regles.Submission{
		SubmittedAt: r.SubmittedAt,
		Seller:      r.Seller.seller(),
		Listing:     r.Listing.listing(),
	}
}

type sellerData struct {
	MemberSince calendarDate `json:"member_since"`
	Suspended   bool         `json:"suspended"`
}

func (s sellerData) seller() regles.Seller {
	return regles.Seller{
		MemberSince: time.Time(s.MemberSince),
		Suspended:   s.Suspended,
	}
}

type calendarDate time.Time

func (d *calendarDate) UnmarshalJSON(raw []byte) error {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return err
	}
	parsed, err := time.Parse(time.DateOnly, text)
	if err != nil {
		return fmt.Errorf("date du compte vendeur : %w", err)
	}
	*d = calendarDate(parsed)
	return nil
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
