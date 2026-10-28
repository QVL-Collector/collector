package annonces

import (
	"fmt"
	"regexp"
	"unicode"
	"unicode/utf8"
)

const (
	MaxTitleLength       = 200
	MaxDescriptionLength = 5000
	MaxPriceCents        = int64(100_000_000)
	MaxShippingFeeCents  = int64(100_000)
	maxCategoryIDLength  = 64
)

var categoryIDFormat = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Field int

const (
	FieldTitle Field = iota + 1
	FieldDescription
	FieldCategory
	FieldPrice
	FieldPriceCurrency
	FieldShippingFee
	FieldShippingFeeCurrency
)

type ViolationCode string

const (
	CodeTooLong             ViolationCode = "trop_long"
	CodeForbiddenCharacter  ViolationCode = "caractere_interdit"
	CodeNegative            ViolationCode = "negatif"
	CodeAboveCeiling        ViolationCode = "au_dela_du_plafond"
	CodeUnsupportedCurrency ViolationCode = "devise_non_prise_en_charge"
	CodeUnknownCategory     ViolationCode = "categorie_inconnue"
)

type Violation struct {
	Field Field
	Code  ViolationCode
}

type ValidationError struct {
	Violations []Violation
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("saisie invalide : %d champ(s) refusé(s)", len(e.Violations))
}

func (c Content) violations() []Violation {
	var found []Violation
	found = appendTextViolations(found, FieldTitle, c.Title, MaxTitleLength, isForbiddenInTitle)
	found = appendTextViolations(found, FieldDescription, c.Description, MaxDescriptionLength, isForbiddenInDescription)
	found = appendCategoryViolations(found, c.CategoryID)
	found = appendMoneyViolations(found, moneyFields{amount: FieldPrice, currency: FieldPriceCurrency}, c.Price, MaxPriceCents)
	found = appendMoneyViolations(found, moneyFields{amount: FieldShippingFee, currency: FieldShippingFeeCurrency}, c.ShippingFee, MaxShippingFeeCents)
	return found
}

func appendTextViolations(found []Violation, field Field, text string, maxLength int, forbidden func(rune) bool) []Violation {
	if utf8.RuneCountInString(text) > maxLength {
		found = append(found, Violation{Field: field, Code: CodeTooLong})
	}
	for _, character := range text {
		if forbidden(character) {
			return append(found, Violation{Field: field, Code: CodeForbiddenCharacter})
		}
	}
	return found
}

func isForbiddenInTitle(character rune) bool {
	return unicode.IsControl(character) || character == utf8.RuneError
}

func isForbiddenInDescription(character rune) bool {
	if character == '\n' || character == '\r' || character == '\t' {
		return false
	}
	return isForbiddenInTitle(character)
}

func appendCategoryViolations(found []Violation, id *CategoryID) []Violation {
	if id == nil {
		return found
	}
	if !id.wellFormed() {
		return append(found, Violation{Field: FieldCategory, Code: CodeUnknownCategory})
	}
	return found
}

func (id CategoryID) wellFormed() bool {
	return len(id) <= maxCategoryIDLength && categoryIDFormat.MatchString(string(id))
}

type moneyFields struct {
	amount   Field
	currency Field
}

func appendMoneyViolations(found []Violation, fields moneyFields, money *Money, ceiling int64) []Violation {
	if money == nil {
		return found
	}
	if money.Cents < 0 {
		found = append(found, Violation{Field: fields.amount, Code: CodeNegative})
	}
	if money.Cents > ceiling {
		found = append(found, Violation{Field: fields.amount, Code: CodeAboveCeiling})
	}
	if money.Currency != Euro {
		found = append(found, Violation{Field: fields.currency, Code: CodeUnsupportedCurrency})
	}
	return found
}

func hasViolationOn(violations []Violation, field Field) bool {
	for _, violation := range violations {
		if violation.Field == field {
			return true
		}
	}
	return false
}
