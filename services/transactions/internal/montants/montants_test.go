package montants

import (
	"crypto/rand"
	"errors"
	"math"
	"math/big"
	"testing"
)

func euros(centimes int64) Money {
	return Money{Cents: centimes, Currency: Euro}
}

func commissionAttendue(total int64) int64 {
	cinqPourCent := new(big.Rat).SetFrac(new(big.Int).Mul(big.NewInt(total), big.NewInt(5)), big.NewInt(100))
	avecDemi := new(big.Rat).Add(cinqPourCent, big.NewRat(1, 2))
	arrondi := new(big.Int).Quo(avecDemi.Num(), avecDemi.Denom())
	return arrondi.Int64()
}

func tirage(t *testing.T, borne int64) int64 {
	t.Helper()
	valeur, err := rand.Int(rand.Reader, big.NewInt(borne))
	if err != nil {
		t.Fatalf("tirage : %v", err)
	}
	return valeur.Int64()
}

func calculer(t *testing.T, prix, frais int64) Breakdown {
	t.Helper()
	detail, err := Compute(euros(prix), euros(frais))
	if err != nil {
		t.Fatalf("Compute(%d, %d) : %v", prix, frais, err)
	}
	return detail
}

func exigerCoherent(t *testing.T, prix, frais int64, detail Breakdown) {
	t.Helper()
	total := prix + frais
	if detail.Price != euros(prix) || detail.ShippingFee != euros(frais) {
		t.Fatalf("prix %+v et frais %+v, attendus %d et %d EUR", detail.Price, detail.ShippingFee, prix, frais)
	}
	if detail.TotalPaid != euros(total) {
		t.Fatalf("total %+v, attendu %d EUR", detail.TotalPaid, total)
	}
	if detail.Commission.Currency != Euro || detail.SellerAmount.Currency != Euro {
		t.Fatalf("devises %s et %s, EUR attendu", detail.Commission.Currency, detail.SellerAmount.Currency)
	}
	if detail.Commission.Cents+detail.SellerAmount.Cents != detail.TotalPaid.Cents {
		t.Fatalf("ENF-01 : commission %d + vendeur %d != total %d", detail.Commission.Cents, detail.SellerAmount.Cents, detail.TotalPaid.Cents)
	}
	if detail.Commission.Cents < 0 || detail.SellerAmount.Cents < 0 {
		t.Fatalf("montant négatif : %+v", detail)
	}
}

func TestUS030AC2CommissionDeCinqPourCentArrondieAuCentimeDemiVersLeHaut(t *testing.T) {
	cas := []struct {
		nom        string
		prix       int64
		frais      int64
		commission int64
	}{
		{"exemple du README : 120 € + 5,90 € donne 6,295 € soit 6,30 €", 12000, 590, 630},
		{"total nul : aucune commission", 0, 0, 0},
		{"un centime : 0,05 c arrondi à 0", 1, 0, 0},
		{"neuf centimes : 0,45 c arrondi à 0", 9, 0, 0},
		{"dix centimes : demi-centime arrondi vers le haut", 10, 0, 1},
		{"dix-neuf centimes : 0,95 c arrondi à 1", 19, 0, 1},
		{"trente centimes : 1,5 c arrondi à 2", 30, 0, 2},
		{"cinquante centimes : 2,5 c arrondi à 3", 50, 0, 3},
		{"frais de port nuls", 4500, 0, 225},
		{"frais seuls, prix nul", 0, 590, 30},
		{"prix et frais demi-centime sur le total seulement", 5, 5, 1},
		{"45 € + 6 € donne 2,55 €", 4500, 600, 255},
		{"19,90 € sans frais : 99,5 c arrondi à 1 €", 1990, 0, 100},
		{"prix maximal du catalogue et frais maximaux", 100_000_000, 100_000, 5_005_000},
		{"grand montant au demi-centime", 1_000_000_010, 0, 50_000_001},
		{"très grand montant", 1_000_000_000_000_000, 0, 50_000_000_000_000},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			detail := calculer(t, c.prix, c.frais)
			exigerCoherent(t, c.prix, c.frais, detail)
			if detail.Commission.Cents != c.commission {
				t.Fatalf("commission %d c, attendue %d c", detail.Commission.Cents, c.commission)
			}
			if detail.SellerAmount.Cents != c.prix+c.frais-c.commission {
				t.Fatalf("montant vendeur %d c, attendu %d c", detail.SellerAmount.Cents, c.prix+c.frais-c.commission)
			}
		})
	}
}

func TestUS030AC2InvariantCommissionPlusVendeurEgalTotalSurTousLesPetitsTotaux(t *testing.T) {
	for total := int64(0); total <= 20_000; total++ {
		for _, frais := range []int64{0, total / 3, total} {
			detail := calculer(t, total-frais, frais)
			exigerCoherent(t, total-frais, frais, detail)
			if detail.Commission.Cents != commissionAttendue(total) {
				t.Fatalf("total %d : commission %d, attendue %d", total, detail.Commission.Cents, commissionAttendue(total))
			}
		}
	}
}

func TestUS030AC2InvariantSurDesMontantsAleatoiresDuCatalogue(t *testing.T) {
	for range 200_000 {
		prix := tirage(t, 1_000_000_000_000)
		frais := tirage(t, 1_000_000)
		detail := calculer(t, prix, frais)
		exigerCoherent(t, prix, frais, detail)
		if detail.Commission.Cents != commissionAttendue(prix+frais) {
			t.Fatalf("total %d : commission %d, attendue %d", prix+frais, detail.Commission.Cents, commissionAttendue(prix+frais))
		}
	}
}

func TestUS030AC2GrandsMontantsAleatoiresExactsOuRefusesJamaisFaux(t *testing.T) {
	for range 200_000 {
		prix := tirage(t, math.MaxInt64/2)
		frais := tirage(t, math.MaxInt64/2)
		exigerExactOuRefuse(t, prix, frais)
	}
}

func TestUS030AC2MontantsAuxLimitesDeInt64ExactsOuRefuses(t *testing.T) {
	cas := []struct {
		nom   string
		prix  int64
		frais int64
	}{
		{"total égal au plus grand int64", math.MaxInt64 - 1, 1},
		{"total juste sous le plus grand int64", math.MaxInt64 / 2, math.MaxInt64 / 2},
		{"prix égal au plus grand int64 sans frais", math.MaxInt64, 0},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			exigerExactOuRefuse(t, c.prix, c.frais)
		})
	}
}

func exigerExactOuRefuse(t *testing.T, prix, frais int64) {
	t.Helper()
	detail, err := Compute(euros(prix), euros(frais))
	if err != nil {
		if !errors.Is(err, ErrAmountTooLarge) {
			t.Fatalf("Compute(%d, %d) : refus %v, ErrAmountTooLarge attendu", prix, frais, err)
		}
		return
	}
	exigerCoherent(t, prix, frais, detail)
	if detail.Commission.Cents != commissionAttendue(prix+frais) {
		t.Fatalf("Compute(%d, %d) : commission %d, attendue %d : calcul faux au lieu d'un refus", prix, frais, detail.Commission.Cents, commissionAttendue(prix+frais))
	}
}

func TestUS030AC6MontantsIncoherentsRefuses(t *testing.T) {
	cas := []struct {
		nom    string
		prix   Money
		frais  Money
		erreur error
	}{
		{"prix négatif", euros(-1), euros(0), ErrNegativeAmount},
		{"frais négatifs", euros(100), euros(-1), ErrNegativeAmount},
		{"plus petit int64", euros(math.MinInt64), euros(0), ErrNegativeAmount},
		{"devises différentes", euros(100), Money{Cents: 10, Currency: "USD"}, ErrCurrencyMismatch},
		{"total hors de int64", euros(math.MaxInt64), euros(1), ErrAmountTooLarge},
		{"deux montants maximaux", euros(math.MaxInt64), euros(math.MaxInt64), ErrAmountTooLarge},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			if _, err := Compute(c.prix, c.frais); !errors.Is(err, c.erreur) {
				t.Fatalf("erreur %v, %v attendue", err, c.erreur)
			}
		})
	}
}
