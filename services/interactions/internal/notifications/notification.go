package notifications

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUnknownIssue    = errors.New("issue de décision inconnue")
	ErrUnknownReason   = errors.New("motif de décision inconnu")
	ErrInvalidRevision = errors.New("révision de décision invalide")
	ErrReasonMismatch  = errors.New("motif incohérent avec l'issue")
)

type NotificationID uuid.UUID

func (id NotificationID) String() string {
	return uuid.UUID(id).String()
}

type SellerID uuid.UUID

func ParseSellerID(subject string) (SellerID, error) {
	parsed, err := uuid.Parse(subject)
	if err != nil {
		return SellerID{}, fmt.Errorf("identifiant de vendeur invalide : %w", err)
	}
	return SellerID(parsed), nil
}

func (id SellerID) String() string {
	return uuid.UUID(id).String()
}

type ArticleID uuid.UUID

func ParseArticleID(raw string) (ArticleID, error) {
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return ArticleID{}, fmt.Errorf("identifiant d'article invalide : %w", err)
	}
	return ArticleID(parsed), nil
}

func (id ArticleID) String() string {
	return uuid.UUID(id).String()
}

type Revision int

const FirstRevision Revision = 1

type Issue string

const (
	IssuePublished Issue = "publie"
	IssueRefused   Issue = "refuse"
	IssueToVerify  Issue = "a_verifier"
)

var issues = []Issue{IssuePublished, IssueRefused, IssueToVerify}

func ParseIssue(raw string) (Issue, error) {
	issue := Issue(raw)
	if !slices.Contains(issues, issue) {
		return "", ErrUnknownIssue
	}
	return issue, nil
}

func (i Issue) RequiresReason() bool {
	return i != IssuePublished
}

type Reason string

const (
	ReasonIncompleteListing Reason = "annonce_incomplete"
	ReasonContactDetected   Reason = "coordonnees_detectees"
	ReasonAccountNotAllowed Reason = "compte_non_autorise"
	ReasonPriceToConfirm    Reason = "prix_a_confirmer"
	ReasonSellerToVerify    Reason = "vendeur_a_verifier"
	ReasonUnknownCategory   Reason = "categorie_inconnue"
	ReasonAbusivePrice      Reason = "prix_abusif"
	ReasonForbiddenItem     Reason = "article_interdit"
	ReasonUnreliableSeller  Reason = "vendeur_non_fiable"
)

var reasons = []Reason{
	ReasonIncompleteListing,
	ReasonContactDetected,
	ReasonAccountNotAllowed,
	ReasonPriceToConfirm,
	ReasonSellerToVerify,
	ReasonUnknownCategory,
	ReasonAbusivePrice,
	ReasonForbiddenItem,
	ReasonUnreliableSeller,
}

func ParseReason(raw string) (Reason, error) {
	reason := Reason(raw)
	if !slices.Contains(reasons, reason) {
		return "", ErrUnknownReason
	}
	return reason, nil
}

type Decision struct {
	MessageID string
	Seller    SellerID
	Article   ArticleID
	Revision  Revision
	Issue     Issue
	Reason    *Reason
	DecidedAt time.Time
}

func (d Decision) Validate() error {
	if d.Revision < FirstRevision {
		return ErrInvalidRevision
	}
	if d.Issue.RequiresReason() != (d.Reason != nil) {
		return ErrReasonMismatch
	}
	return nil
}

type Notification struct {
	ID        NotificationID
	Article   ArticleID
	Revision  Revision
	Issue     Issue
	Reason    *Reason
	DecidedAt time.Time
}

const (
	DefaultPageSize = 20
	MaxPageSize     = 50
	MaxPageNumber   = 10_000
)

type PageRequest struct {
	Number int
	Size   int
}

func (p PageRequest) Offset() int {
	return (p.Number - 1) * p.Size
}

type Page struct {
	Notifications []Notification
	Total         int
}
