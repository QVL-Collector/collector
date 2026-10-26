package controle

import (
	"time"

	"collector/libs/platform/message"
)

const (
	RequestExchange = "controle.demandes"
	ResultQueue     = "pdm.resultats-controle"
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

type sellerData struct {
	AccountID   string `json:"account_id"`
	MemberSince string `json:"member_since"`
	Suspended   bool   `json:"suspended"`
}

type listingData struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	CategoryCode *string `json:"category_code"`
	PriceCents   *int64  `json:"price_cents"`
	PhotoCount   int     `json:"photo_count"`
}

type resultData struct {
	ArticleID    string  `json:"article_id"`
	Revision     int     `json:"revision"`
	RulesVersion int     `json:"rules_version"`
	Issue        string  `json:"issue"`
	Reason       *string `json:"reason"`
}
