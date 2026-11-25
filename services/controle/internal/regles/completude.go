package regles

import "strings"

const minimumPhotos = 1

func complete(listing Listing) bool {
	return filled(listing.Title) &&
		filled(listing.Description) &&
		listing.CategoryCode != nil && filled(string(*listing.CategoryCode)) &&
		listing.PriceCents != nil &&
		listing.PhotoCount >= minimumPhotos
}

func filled(text string) bool {
	return strings.TrimSpace(text) != ""
}
