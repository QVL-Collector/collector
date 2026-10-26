package decision

import (
	"errors"
	"fmt"
	"time"

	"collector/sandbox/xp02/internal/ident"
)

const (
	EventType       = "publication.decidee"
	ContractVersion = 1
)

var ErrInvalid = errors.New("décision de publication invalide")

type Issue string

const (
	Published Issue = "publie"
	Refused   Issue = "refuse"
	ToVerify  Issue = "a_verifier"
)

func (i Issue) valid() bool {
	switch i {
	case Published, Refused, ToVerify:
		return true
	}
	return false
}

type Decision struct {
	ID        string    `json:"decision_id"`
	ArticleID string    `json:"article_id"`
	Revision  int       `json:"revision"`
	Issue     Issue     `json:"issue"`
	DecidedAt time.Time `json:"decided_at"`
}

func New(articleID string, revision int, issue Issue, decidedAt time.Time) (Decision, error) {
	switch {
	case !ident.Valid(articleID):
		return Decision{}, fmt.Errorf("%w : identifiant d'article %q", ErrInvalid, articleID)
	case revision < 1:
		return Decision{}, fmt.Errorf("%w : révision %d", ErrInvalid, revision)
	case !issue.valid():
		return Decision{}, fmt.Errorf("%w : issue %q", ErrInvalid, issue)
	}
	id, err := ident.New()
	if err != nil {
		return Decision{}, err
	}
	return Decision{
		ID:        id,
		ArticleID: articleID,
		Revision:  revision,
		Issue:     issue,
		DecidedAt: decidedAt.UTC(),
	}, nil
}
