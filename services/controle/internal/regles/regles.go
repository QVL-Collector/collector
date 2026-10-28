package regles

import "time"

const RulesVersion = 3

type Issue string

const (
	IssuePublished Issue = "publie"
	IssueRefused   Issue = "refuse"
	IssueToVerify  Issue = "a_verifier"
)

type Reason string

const (
	ReasonUnauthorizedAccount Reason = "compte_non_autorise"
	ReasonIncompleteListing   Reason = "annonce_incomplete"
	ReasonContactDetected     Reason = "coordonnees_detectees"
	ReasonUnknownCategory     Reason = "categorie_inconnue"
	ReasonPriceToConfirm      Reason = "prix_a_confirmer"
	ReasonSellerToVerify      Reason = "vendeur_a_verifier"
)

type CategoryCode string

type Listing struct {
	Title        string
	Description  string
	CategoryCode *CategoryCode
	PriceCents   *int64
}

type Seller struct {
	MemberSince time.Time
	Suspended   bool
}

type Submission struct {
	SubmittedAt time.Time
	Seller      Seller
	Listing     Listing
}

type Verdict struct {
	Issue  Issue
	Reason Reason
}

func (v Verdict) HasReason() bool {
	return v.Reason != ""
}

func refused(reason Reason) Verdict {
	return Verdict{Issue: IssueRefused, Reason: reason}
}

func toVerify(reason Reason) Verdict {
	return Verdict{Issue: IssueToVerify, Reason: reason}
}

type Rules struct {
	catalogue Catalogue
	seniority SeniorityPolicy
}

func New(catalogue Catalogue, seniority SeniorityPolicy) Rules {
	return Rules{catalogue: catalogue.clone(), seniority: seniority}
}

func (r Rules) Evaluate(submission Submission) Verdict {
	listing := submission.Listing
	switch {
	case !authorizedToSell(submission.Seller):
		return refused(ReasonUnauthorizedAccount)
	case !complete(listing):
		return refused(ReasonIncompleteListing)
	case containsContactDetails(listing):
		return refused(ReasonContactDetected)
	}
	priceRange, known := r.catalogue.priceRangeOf(*listing.CategoryCode)
	price := *listing.PriceCents
	switch {
	case !known:
		return toVerify(ReasonUnknownCategory)
	case !priceRange.accepts(price):
		return toVerify(ReasonPriceToConfirm)
	case r.seniority.isNewcomer(submission.Seller, submission.SubmittedAt) && priceRange.exceedsVigilance(price):
		return toVerify(ReasonSellerToVerify)
	}
	return Verdict{Issue: IssuePublished}
}
