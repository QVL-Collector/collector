type DemoArticlePayload = {
  readonly id: string;
  readonly titre: string;
  readonly prix: { readonly montantCentimes: number; readonly devise: string };
  readonly categorie: { readonly id: string; readonly libelle: string };
};

const categories = {
  vinyles: { id: 'vinyles', libelle: 'Vinyles' },
  montres: { id: 'montres', libelle: 'Montres' },
  jouets: { id: 'jouets-anciens', libelle: 'Jouets anciens' },
  cartes: { id: 'cartes-a-collectionner', libelle: 'Cartes à collectionner' },
  bd: { id: 'bandes-dessinees', libelle: 'Bandes dessinées' },
  monnaies: { id: 'monnaies', libelle: 'Monnaies' },
} as const;

function demoArticle(
  index: number,
  titre: string,
  montantCentimes: number,
  categorie: DemoArticlePayload['categorie'],
): DemoArticlePayload {
  return {
    id: `demo-${String(index).padStart(3, '0')}`,
    titre,
    prix: { montantCentimes, devise: 'EUR' },
    categorie,
  };
}

export const demoArticles: readonly DemoArticlePayload[] = [
  demoArticle(1, 'Abbey Road, pressage original 1969', 24000, categories.vinyles),
  demoArticle(2, 'Montre de gousset en argent, mouvement Lip', 18500, categories.montres),
  demoArticle(3, 'Train mécanique Hornby en tôle lithographiée', 12900, categories.jouets),
  demoArticle(4, 'Carte Dracaufeu édition de base, état excellent', 45000, categories.cartes),
  demoArticle(5, 'Tintin au Tibet, édition originale 1960', 32000, categories.bd),
  demoArticle(6, 'Pièce de 5 francs Semeuse en argent, 1962', 1800, categories.monnaies),
  demoArticle(
    7,
    '<script>alert("collector")</script> Pochette dédicacée',
    9900,
    categories.vinyles,
  ),
  demoArticle(8, 'Chronographe automatique des années 1970', 61000, categories.montres),
  demoArticle(9, 'Poupée Bella en celluloïd, robe d’origine', 8700, categories.jouets),
  demoArticle(10, '<img src=x onerror=alert(1)> Lot de cartes rares', 15000, categories.cartes),
  demoArticle(11, 'Astérix le Gaulois, tirage de 1961', 27500, categories.bd),
  demoArticle(12, 'Napoléon III, 20 francs or, 1859', 39000, categories.monnaies),
  demoArticle(13, 'Kind of Blue, réédition audiophile', 6500, categories.vinyles),
  demoArticle(14, 'Montre plongée vintage, lunette tournante', 42000, categories.montres),
  demoArticle(15, 'Voiture miniature Dinky Toys au 1/43, boîte d’origine', 7400, categories.jouets),
  demoArticle(16, 'Coffret de cartes de 1996, scellé', 52000, categories.cartes),
  demoArticle(17, 'Spirou, numéro spécial anniversaire', 3900, categories.bd),
  demoArticle(18, 'Écu Louis XV en argent, 1726', 68000, categories.monnaies),
  demoArticle(19, 'Thriller, picture disc numéroté', 11000, categories.vinyles),
  demoArticle(20, 'Montre de poche ferroviaire, cadran émaillé', 23000, categories.montres),
  demoArticle(21, 'Robot jouet à remontoir des années 1960', 16500, categories.jouets),
  demoArticle(22, 'Carte de joueur de football 1958, signée', 98000, categories.cartes),
  demoArticle(23, 'Blake et Mortimer, La Marque jaune, 1956', 41000, categories.bd),
  demoArticle(24, 'Pièce de 10 euros commémorative, épreuve', 2500, categories.monnaies),
  demoArticle(25, 'Disque 78 tours de jazz, label rare', 5400, categories.vinyles),
  demoArticle(
    26,
    'Toupie en bois peint de collection, fabrication artisanale',
    2900,
    categories.jouets,
  ),
];
