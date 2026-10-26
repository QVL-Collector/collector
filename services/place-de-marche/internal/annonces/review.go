package annonces

import (
	"errors"
	"slices"
	"time"
)

var (
	ErrUnknownReviewReason = errors.New("motif de contrôle inconnu")
	ErrUnknownReviewIssue  = errors.New("issue de contrôle inconnue")
)

type ReviewReason string

const (
	ReasonIncompleteListing ReviewReason = "annonce_incomplete"
	ReasonContactDetected   ReviewReason = "coordonnees_detectees"
	ReasonAccountNotAllowed ReviewReason = "compte_non_autorise"
	ReasonPriceToConfirm    ReviewReason = "prix_a_confirmer"
	ReasonSellerToVerify    ReviewReason = "vendeur_a_verifier"
	ReasonUnknownCategory   ReviewReason = "categorie_inconnue"
)

var reviewReasons = []ReviewReason{
	ReasonIncompleteListing,
	ReasonContactDetected,
	ReasonAccountNotAllowed,
	ReasonPriceToConfirm,
	ReasonSellerToVerify,
	ReasonUnknownCategory,
}

var reviewIssues = map[string]Status{
	"publie":     StatusPublished,
	"refuse":     StatusRefused,
	"a_verifier": StatusToVerify,
}

func ParseReviewReason(raw string) (ReviewReason, error) {
	reason := ReviewReason(raw)
	if !slices.Contains(reviewReasons, reason) {
		return "", ErrUnknownReviewReason
	}
	return reason, nil
}

func ParseReviewIssue(raw string) (Status, error) {
	status, known := reviewIssues[raw]
	if !known {
		return "", ErrUnknownReviewIssue
	}
	return status, nil
}

type SellerStatus struct {
	MemberSince time.Time
	Suspended   bool
}

type ReviewRequest struct {
	Article     Article
	Seller      SellerStatus
	SubmittedAt time.Time
	PhotoCount  int
}

type ReviewOutcome struct {
	MessageID    string
	Article      ArticleID
	Revision     Revision
	Status       Status
	Reason       *ReviewReason
	RulesVersion int
}
