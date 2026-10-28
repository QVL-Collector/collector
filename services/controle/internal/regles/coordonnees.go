package regles

import "collector/services/controle/internal/coordonnees"

func containsContactDetails(listing Listing) bool {
	return coordonnees.Detected(listing.Title) || coordonnees.Detected(listing.Description)
}
