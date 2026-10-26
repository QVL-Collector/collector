package annonces

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrArticleNotFound    = errors.New("article introuvable")
	ErrStaleRevision      = errors.New("révision périmée")
	ErrArticleNotEditable = errors.New("article non modifiable dans son état")
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

type Revision int

const FirstRevision Revision = 1

type Status string

const StatusDraft Status = "brouillon"

func (s Status) Editable() bool {
	return s == StatusDraft
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
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
