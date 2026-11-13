package diffusion

import (
	"time"

	"collector/libs/platform/message"
)

const (
	EventExchange      = "pdm.evenements"
	decisionRoutingKey = "publication.decidee.v1"
	contractVersion    = 1
)

var decisionContract = message.Contract{Type: "publication.decidee", Version: contractVersion}

type decisionData struct {
	ArticleID string    `json:"article_id"`
	Revision  int       `json:"revision"`
	SellerID  string    `json:"seller_id"`
	Issue     string    `json:"issue"`
	Reason    *string   `json:"reason,omitempty"`
	DecidedAt time.Time `json:"decided_at"`
}
