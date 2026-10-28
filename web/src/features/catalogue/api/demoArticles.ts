type DemoAmount = { readonly montantCentimes: number; readonly devise: string };

type DemoCategoryPayload = { readonly id: string; readonly libelle: string };

export type DemoArticlePayload = {
  readonly id: string;
  readonly titre: string;
  readonly prix: DemoAmount;
  readonly categorie: DemoCategoryPayload;
  readonly description: string;
  readonly fraisDePort: DemoAmount | null;
};

type DemoArticleContent = {
  readonly titre: string;
  readonly prix: number;
  readonly categorie: DemoCategoryPayload;
  readonly description: string;
  readonly fraisDePort: number | null;
};

const categories = {
  vinyles: { id: 'vinyles', libelle: 'Vinyles' },
  montres: { id: 'montres', libelle: 'Montres' },
  jouets: { id: 'jouets-anciens', libelle: 'Jouets anciens' },
  cartes: { id: 'cartes-a-collectionner', libelle: 'Cartes à collectionner' },
  bd: { id: 'bandes-dessinees', libelle: 'Bandes dessinées' },
  monnaies: { id: 'monnaies', libelle: 'Monnaies' },
} as const;

export const demoCategories: readonly DemoCategoryPayload[] = [...Object.values(categories)].sort(
  (first, second) => first.libelle.localeCompare(second.libelle, 'fr'),
);

function euros(montantCentimes: number): DemoAmount {
  return { montantCentimes, devise: 'EUR' };
}

function demoArticle(index: number, content: DemoArticleContent): DemoArticlePayload {
  return {
    id: `demo-${String(index).padStart(3, '0')}`,
    titre: content.titre,
    prix: euros(content.prix),
    categorie: content.categorie,
    description: content.description,
    fraisDePort: content.fraisDePort === null ? null : euros(content.fraisDePort),
  };
}

const contents: readonly DemoArticleContent[] = [
  {
    titre: 'Abbey Road, pressage original 1969',
    prix: 24000,
    categorie: categories.vinyles,
    description:
      'Pressage britannique Apple, pochette ouvrante d’origine.\nDisque lavé, quelques craquements discrets sur la face B.',
    fraisDePort: 900,
  },
  {
    titre: 'Montre de gousset en argent, mouvement Lip',
    prix: 18500,
    categorie: categories.montres,
    description: 'Boîtier poinçonné, révisée en 2024, fonctionne parfaitement.',
    fraisDePort: 1200,
  },
  {
    titre: 'Train mécanique Hornby en tôle lithographiée',
    prix: 12900,
    categorie: categories.jouets,
    description: 'Locomotive, deux wagons et rails d’origine. Clé de remontage fournie.',
    fraisDePort: null,
  },
  {
    titre: 'Carte Dracaufeu édition de base, état excellent',
    prix: 45000,
    categorie: categories.cartes,
    description: 'Conservée sous étui rigide depuis l’achat, coins nets.',
    fraisDePort: 500,
  },
  {
    titre: 'Tintin au Tibet, édition originale 1960',
    prix: 32000,
    categorie: categories.bd,
    description: 'Dos toilé rouge, intérieur frais, sans annotation.',
    fraisDePort: 800,
  },
  {
    titre: 'Pièce de 5 francs Semeuse en argent, 1962',
    prix: 1800,
    categorie: categories.monnaies,
    description: 'Belle patine, livrée sous capsule.',
    fraisDePort: 300,
  },
  {
    titre: '<script>alert("collector")</script> Pochette dédicacée',
    prix: 9900,
    categorie: categories.vinyles,
    description:
      '<b>Dédicace</b> au feutre <script>alert("fiche")</script> sur la pochette intérieure.',
    fraisDePort: 700,
  },
  {
    titre: 'Chronographe automatique des années 1970',
    prix: 61000,
    categorie: categories.montres,
    description: 'Cadran panda, bracelet cuir neuf, boîte non d’origine.',
    fraisDePort: 1500,
  },
  {
    titre: 'Poupée Bella en celluloïd, robe d’origine',
    prix: 8700,
    categorie: categories.jouets,
    description: 'Yeux dormeurs fonctionnels, légère décoloration du tissu.',
    fraisDePort: 900,
  },
  {
    titre: '<img src=x onerror=alert(1)> Lot de cartes rares',
    prix: 15000,
    categorie: categories.cartes,
    description: 'Lot de vingt cartes, liste détaillée sur demande.',
    fraisDePort: 400,
  },
  {
    titre: 'Astérix le Gaulois, tirage de 1961',
    prix: 27500,
    categorie: categories.bd,
    description: 'Premier tirage, dos légèrement frotté.',
    fraisDePort: 800,
  },
  {
    titre: 'Napoléon III, 20 francs or, 1859',
    prix: 39000,
    categorie: categories.monnaies,
    description: 'Atelier de Paris, très bel état.',
    fraisDePort: null,
  },
  {
    titre: 'Kind of Blue, réédition audiophile',
    prix: 6500,
    categorie: categories.vinyles,
    description: 'Pressage 180 g, écouté deux fois.',
    fraisDePort: 900,
  },
  {
    titre: 'Montre plongée vintage, lunette tournante',
    prix: 42000,
    categorie: categories.montres,
    description: 'Étanchéité non garantie, verre d’origine.',
    fraisDePort: 1500,
  },
  {
    titre: 'Voiture miniature Dinky Toys au 1/43, boîte d’origine',
    prix: 7400,
    categorie: categories.jouets,
    description: 'Peinture d’origine, boîte complète avec rabats.',
    fraisDePort: 600,
  },
  {
    titre: 'Coffret de cartes de 1996, scellé',
    prix: 52000,
    categorie: categories.cartes,
    description: 'Film d’origine intact, aucune trace d’ouverture.',
    fraisDePort: 1000,
  },
  {
    titre: 'Spirou, numéro spécial anniversaire',
    prix: 3900,
    categorie: categories.bd,
    description: '',
    fraisDePort: 400,
  },
  {
    titre: 'Écu Louis XV en argent, 1726',
    prix: 68000,
    categorie: categories.monnaies,
    description: 'Exemplaire certifié, certificat joint.',
    fraisDePort: 1200,
  },
  {
    titre: 'Thriller, picture disc numéroté',
    prix: 11000,
    categorie: categories.vinyles,
    description: 'Numéro 0427, jamais joué.',
    fraisDePort: 900,
  },
  {
    titre: 'Montre de poche ferroviaire, cadran émaillé',
    prix: 23000,
    categorie: categories.montres,
    description: 'Cadran sans fêle, aiguilles d’origine.',
    fraisDePort: 1200,
  },
  {
    titre: 'Robot jouet à remontoir des années 1960',
    prix: 16500,
    categorie: categories.jouets,
    description: 'Marche et clignote, mécanisme d’origine.',
    fraisDePort: 900,
  },
  {
    titre: 'Carte de joueur de football 1958, signée',
    prix: 98000,
    categorie: categories.cartes,
    description: 'Signature authentifiée, carte sous coque scellée.',
    fraisDePort: 600,
  },
  {
    titre: 'Blake et Mortimer, La Marque jaune, 1956',
    prix: 41000,
    categorie: categories.bd,
    description: 'Édition originale, coiffes intactes.',
    fraisDePort: 800,
  },
  {
    titre: 'Pièce de 10 euros commémorative, épreuve',
    prix: 2500,
    categorie: categories.monnaies,
    description: 'Qualité belle épreuve, coffret et certificat.',
    fraisDePort: 300,
  },
  {
    titre: 'Disque 78 tours de jazz, label rare',
    prix: 5400,
    categorie: categories.vinyles,
    description: 'Gomme-laque, pochette papier d’époque.',
    fraisDePort: 900,
  },
  {
    titre: 'Toupie en bois peint de collection, fabrication artisanale',
    prix: 2900,
    categorie: categories.jouets,
    description: 'Pièce unique tournée à la main.',
    fraisDePort: null,
  },
];

export const demoArticles: readonly DemoArticlePayload[] = contents.map((content, index) =>
  demoArticle(index + 1, content),
);
