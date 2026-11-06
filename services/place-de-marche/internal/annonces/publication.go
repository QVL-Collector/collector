package annonces

import "time"

type PublicationDecision struct {
	Article   ArticleID
	Revision  Revision
	Seller    SellerID
	Outcome   Status
	Reason    *ReviewReason
	DecidedAt time.Time
}

func DecisionOn(article Article) PublicationDecision {
	return PublicationDecision{
		Article:   article.ID,
		Revision:  article.Revision,
		Seller:    article.Seller,
		Outcome:   article.Status,
		Reason:    article.Reason,
		DecidedAt: article.UpdatedAt,
	}
}

func (o ReviewOutcome) DecisionFor(seller SellerID, decidedAt time.Time) PublicationDecision {
	return PublicationDecision{
		Article:   o.Article,
		Revision:  o.Revision,
		Seller:    seller,
		Outcome:   o.Status,
		Reason:    o.Reason,
		DecidedAt: decidedAt,
	}
}
