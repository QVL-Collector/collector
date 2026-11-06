package canal

import (
	"time"

	"collector/libs/platform/message"
	"collector/services/interactions/internal/notifications"
)

const (
	DecisionQueue   = "interactions.publication-decidee"
	contractVersion = 1
)

var decisionContract = message.Contract{Type: "publication.decidee", Version: contractVersion}

type decisionData struct {
	ArticleID string    `json:"article_id"`
	Revision  int       `json:"revision"`
	SellerID  string    `json:"seller_id"`
	Issue     string    `json:"issue"`
	Reason    *string   `json:"reason"`
	DecidedAt time.Time `json:"decided_at"`
}

func toDecision(messageID string, data decisionData) (notifications.Decision, error) {
	article, err := notifications.ParseArticleID(data.ArticleID)
	if err != nil {
		return notifications.Decision{}, err
	}
	seller, err := notifications.ParseSellerID(data.SellerID)
	if err != nil {
		return notifications.Decision{}, err
	}
	issue, err := notifications.ParseIssue(data.Issue)
	if err != nil {
		return notifications.Decision{}, err
	}
	reason, err := reasonOf(data.Reason)
	if err != nil {
		return notifications.Decision{}, err
	}
	decision := notifications.Decision{
		MessageID: messageID,
		Seller:    seller,
		Article:   article,
		Revision:  notifications.Revision(data.Revision),
		Issue:     issue,
		Reason:    reason,
		DecidedAt: data.DecidedAt,
	}
	return decision, decision.Validate()
}

func reasonOf(raw *string) (*notifications.Reason, error) {
	if raw == nil {
		return nil, nil
	}
	reason, err := notifications.ParseReason(*raw)
	if err != nil {
		return nil, err
	}
	return &reason, nil
}
