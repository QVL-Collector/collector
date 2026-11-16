package montants

import (
	"errors"
	"math"
)

type Currency string

const Euro Currency = "EUR"

const (
	commissionBasisPoints  = 500
	basisPointsPerUnit     = 10_000
	halfBasisPointsUnit    = basisPointsPerUnit / 2
	maxCommissionableCents = (math.MaxInt64 - halfBasisPointsUnit) / commissionBasisPoints
)

var (
	ErrNegativeAmount   = errors.New("montant négatif")
	ErrCurrencyMismatch = errors.New("devises différentes")
	ErrAmountTooLarge   = errors.New("montant hors limites")
)

type Money struct {
	Cents    int64
	Currency Currency
}

type Breakdown struct {
	Price        Money
	ShippingFee  Money
	TotalPaid    Money
	Commission   Money
	SellerAmount Money
}

func Compute(price, shippingFee Money) (Breakdown, error) {
	if price.Currency != shippingFee.Currency {
		return Breakdown{}, ErrCurrencyMismatch
	}
	if price.Cents < 0 || shippingFee.Cents < 0 {
		return Breakdown{}, ErrNegativeAmount
	}
	if price.Cents > maxCommissionableCents-shippingFee.Cents {
		return Breakdown{}, ErrAmountTooLarge
	}
	totalPaid := price.Cents + shippingFee.Cents
	commission := commissionCents(totalPaid)
	currency := price.Currency
	return Breakdown{
		Price:        price,
		ShippingFee:  shippingFee,
		TotalPaid:    Money{Cents: totalPaid, Currency: currency},
		Commission:   Money{Cents: commission, Currency: currency},
		SellerAmount: Money{Cents: totalPaid - commission, Currency: currency},
	}, nil
}

func commissionCents(totalPaidCents int64) int64 {
	return (totalPaidCents*commissionBasisPoints + halfBasisPointsUnit) / basisPointsPerUnit
}
