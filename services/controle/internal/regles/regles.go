package regles

import "strings"

const RulesVersion = 1

type Issue string

const (
	IssuePublished Issue = "publie"
	IssueRefused   Issue = "refuse"
	IssueToVerify  Issue = "a_verifier"
)

type Reason string

const (
	ReasonIncompleteListing Reason = "annonce_incomplete"
	ReasonUnknownCategory   Reason = "categorie_inconnue"
)

type CategoryCode string

type Listing struct {
	Title        string
	Description  string
	CategoryCode *CategoryCode
	PriceCents   *int64
}

type Verdict struct {
	Issue  Issue
	Reason Reason
}

func (v Verdict) HasReason() bool {
	return v.Reason != ""
}

type Rules struct {
	categories map[CategoryCode]struct{}
}

func New(categories []CategoryCode) Rules {
	known := make(map[CategoryCode]struct{}, len(categories))
	for _, code := range categories {
		known[code] = struct{}{}
	}
	return Rules{categories: known}
}

func (r Rules) Evaluate(listing Listing) Verdict {
	if !complete(listing) {
		return Verdict{Issue: IssueRefused, Reason: ReasonIncompleteListing}
	}
	if !r.knows(*listing.CategoryCode) {
		return Verdict{Issue: IssueToVerify, Reason: ReasonUnknownCategory}
	}
	return Verdict{Issue: IssuePublished}
}

func (r Rules) knows(code CategoryCode) bool {
	_, known := r.categories[code]
	return known
}

func complete(listing Listing) bool {
	return filled(listing.Title) &&
		filled(listing.Description) &&
		listing.CategoryCode != nil && filled(string(*listing.CategoryCode)) &&
		listing.PriceCents != nil
}

func filled(text string) bool {
	return strings.TrimSpace(text) != ""
}
