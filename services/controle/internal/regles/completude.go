package regles

import "strings"

func complete(listing Listing) bool {
	return filled(listing.Title) &&
		filled(listing.Description) &&
		listing.CategoryCode != nil && filled(string(*listing.CategoryCode)) &&
		listing.PriceCents != nil
}

func filled(text string) bool {
	return strings.TrimSpace(text) != ""
}
