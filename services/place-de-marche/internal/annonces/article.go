package annonces

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

var (
	ErrArticleNotFound       = errors.New("article introuvable")
	ErrStaleRevision         = errors.New("révision périmée")
	ErrArticleNotEditable    = errors.New("article non modifiable dans son état")
	ErrArticleNotSubmittable = errors.New("article non soumettable dans son état")
)

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

type Revision int

const FirstRevision Revision = 1

type Status string

const (
	StatusDraft       Status = "brouillon"
	StatusUnderReview Status = "en_controle"
	StatusPublished   Status = "publie"
	StatusRefused     Status = "refuse"
	StatusToVerify    Status = "a_verifier"
)

var sellerHeldStatuses = []Status{StatusDraft, StatusRefused}

func SellerHeldStatuses() []Status {
	return slices.Clone(sellerHeldStatuses)
}

func (s Status) Editable() bool {
	return slices.Contains(sellerHeldStatuses, s)
}

func (s Status) Submittable() bool {
	return slices.Contains(sellerHeldStatuses, s)
}

type Currency string

const Euro Currency = "EUR"

type Money struct {
	Cents    int64
	Currency Currency
}

type CategoryID string

type Category struct {
	ID    CategoryID
	Label string
}

type Content struct {
	Title       string
	Description string
	CategoryID  *CategoryID
	Price       *Money
	ShippingFee *Money
}

type Article struct {
	ID          ArticleID
	Seller      SellerID
	Status      Status
	Revision    Revision
	Title       string
	Description string
	Category    *Category
	Price       *Money
	ShippingFee *Money
	Reason      *ReviewReason
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
