package annonces

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
)

var (
	ErrArticleNotToVerify   = errors.New("article qui n'est plus à vérifier")
	ErrUnknownVerdict       = errors.New("décision inconnue")
	ErrUnknownRefusalReason = errors.New("motif de refus inconnu")
)

const (
	ReasonAbusivePrice     ReviewReason = "prix_abusif"
	ReasonForbiddenItem    ReviewReason = "article_interdit"
	ReasonUnreliableSeller ReviewReason = "vendeur_non_fiable"
)

var refusalReasons = []ReviewReason{
	ReasonIncompleteListing,
	ReasonContactDetected,
	ReasonAbusivePrice,
	ReasonForbiddenItem,
	ReasonUnreliableSeller,
}

func ParseRefusalReason(raw string) (ReviewReason, error) {
	reason := ReviewReason(raw)
	if !slices.Contains(refusalReasons, reason) {
		return "", ErrUnknownRefusalReason
	}
	return reason, nil
}

type AdminID uuid.UUID

func ParseAdminID(subject string) (AdminID, error) {
	parsed, err := uuid.Parse(subject)
	if err != nil {
		return AdminID{}, fmt.Errorf("identifiant d'administrateur invalide : %w", err)
	}
	return AdminID(parsed), nil
}

func (id AdminID) String() string {
	return uuid.UUID(id).String()
}

type Verdict string

const (
	VerdictPublish Verdict = "publier"
	VerdictRefuse  Verdict = "refuser"
)

var verdictOutcomes = map[Verdict]Status{
	VerdictPublish: StatusPublished,
	VerdictRefuse:  StatusRefused,
}

func ParseVerdict(raw string) (Verdict, error) {
	verdict := Verdict(raw)
	if _, known := verdictOutcomes[verdict]; !known {
		return "", ErrUnknownVerdict
	}
	return verdict, nil
}

func (v Verdict) Outcome() Status {
	return verdictOutcomes[v]
}

func (v Verdict) RequiresReason() bool {
	return v == VerdictRefuse
}

type Decision struct {
	Admin    AdminID
	Article  ArticleID
	Revision Revision
	Verdict  Verdict
	Reason   *ReviewReason
}

func DecisionConflict(status Status, current, examined Revision) error {
	if current == examined && status != StatusToVerify {
		return ErrArticleNotToVerify
	}
	return ErrStaleRevision
}

func (s *Service) ArticlesToVerify(ctx context.Context, page PageRequest) (ArticlePage, error) {
	return s.articles.ListToVerify(ctx, page)
}

func (s *Service) Decide(ctx context.Context, decision Decision) (Article, error) {
	return s.articles.Decide(ctx, decision)
}
