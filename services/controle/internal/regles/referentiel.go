package regles

var referenceCategories = []CategoryCode{
	"bandes-dessinees",
	"cartes-a-collectionner",
	"figurines",
	"jouets-anciens",
	"monnaies",
	"montres",
	"vinyles",
}

func Reference() Rules {
	return New(referenceCategories)
}
