package regles

func authorizedToSell(seller Seller) bool {
	return !seller.Suspended
}
