import type { Langue } from './langues';

export type Textes = {
  readonly nomLangue: string;
  readonly retourCatalogue: string;
  readonly catalogue: string;
  readonly filtreCategorie: string;
  readonly annonceCategorieMontres: RegExp;
  readonly vendeurParticulier: string;
  readonly seConnecter: string;
  readonly seDeconnecter: string;
  readonly roleVendeur: string;
  readonly espaceVendeur: string;
  readonly mesArticles: string;
  readonly nouvelArticle: string;
  readonly retourMesArticles: string;
  readonly champTitre: string;
  readonly champDescription: string;
  readonly champCategorie: string;
  readonly champPrix: string;
  readonly champFraisDePort: string;
  readonly prixSaisi: string;
  readonly creerBrouillon: string;
  readonly etatAnnonce: string;
  readonly brouillon: string;
  readonly enControle: string;
  readonly publie: string;
  readonly refuse: string;
  readonly soumettre: string;
  readonly avertissementRetrait: string;
  readonly enregistrerEtRenvoyer: string;
  readonly renvoyeAuControle: RegExp;
  readonly motifIncomplet: string;
  readonly erreurMontant: string;
  readonly erreurFormulaire: string;
  readonly accesReserve: string;
  readonly echecConnexion: string;
  readonly echecDemarrage: string;
  readonly pageIntrouvable: string;
  readonly articleIntrouvable: string;
  readonly categorieInconnue: string;
  readonly articlesAVerifier: string;
  readonly publier: string;
  readonly refuser: string;
  readonly motifDuRefus: string;
  readonly motifManquant: string;
  readonly confirmerRefus: string;
  readonly motifPrixAbusif: string;
  readonly motifControlePrix: string;
  readonly articlePublie: RegExp;
  readonly articleRefuse: RegExp;
  readonly conflitAdmin: string;
};

const fr: Textes = {
  nomLangue: 'FR, Français',
  retourCatalogue: 'Collector, retour au catalogue',
  catalogue: 'Catalogue',
  filtreCategorie: 'Catégorie',
  annonceCategorieMontres: /articles? publiés? dans la catégorie Montres/,
  vendeurParticulier: 'Vendeur particulier',
  seConnecter: 'Se connecter',
  seDeconnecter: 'Se déconnecter',
  roleVendeur: 'Vendeur',
  espaceVendeur: 'Espace vendeur',
  mesArticles: 'Mes articles',
  nouvelArticle: 'Nouvel article',
  retourMesArticles: 'Retour à mes articles',
  champTitre: 'Titre',
  champDescription: 'Description',
  champCategorie: 'Catégorie',
  champPrix: 'Prix (€)',
  champFraisDePort: 'Frais de port (€)',
  prixSaisi: '42,50',
  creerBrouillon: 'Créer le brouillon',
  etatAnnonce: 'État de l’annonce',
  brouillon: 'Brouillon',
  enControle: 'En contrôle',
  publie: 'Publié',
  refuse: 'Refusé',
  soumettre: 'Soumettre au contrôle',
  avertissementRetrait: 'L’annonce sera retirée du catalogue',
  enregistrerEtRenvoyer: 'Enregistrer et renvoyer au contrôle',
  renvoyeAuControle: /révision 3 : l’annonce repasse au contrôle/,
  motifIncomplet: 'Motif : Annonce incomplète',
  erreurMontant: 'Valeur invalide. Exemple de montant : 12,50.',
  erreurFormulaire: 'Le formulaire contient 1 erreur : corrigez le champ signalé.',
  accesReserve: 'Accès réservé',
  echecConnexion: 'La connexion n’a pas abouti',
  echecDemarrage: 'L’application n’a pas pu démarrer',
  pageIntrouvable: 'Page introuvable',
  articleIntrouvable: 'Article introuvable',
  categorieInconnue: 'Cette catégorie n’existe pas',
  articlesAVerifier: 'Articles à vérifier',
  publier: 'Publier',
  refuser: 'Refuser',
  motifDuRefus: 'Motif du refus',
  motifManquant: 'Choisissez un motif de refus.',
  confirmerRefus: 'Confirmer le refus',
  motifPrixAbusif: 'Prix abusif',
  motifControlePrix: 'Motif du contrôle : Prix à confirmer',
  articlePublie: /a été publié\./,
  articleRefuse: /a été refusé\./,
  conflitAdmin: 'L’article a changé, rechargez la liste',
};

const en: Textes = {
  nomLangue: 'EN, English',
  retourCatalogue: 'Collector, back to the catalogue',
  catalogue: 'Catalogue',
  filtreCategorie: 'Category',
  annonceCategorieMontres: /published items? in the Montres category/,
  vendeurParticulier: 'Private seller',
  seConnecter: 'Sign in',
  seDeconnecter: 'Sign out',
  roleVendeur: 'Seller',
  espaceVendeur: 'Seller area',
  mesArticles: 'My items',
  nouvelArticle: 'New item',
  retourMesArticles: 'Back to my items',
  champTitre: 'Title',
  champDescription: 'Description',
  champCategorie: 'Category',
  champPrix: 'Price (€)',
  champFraisDePort: 'Shipping cost (€)',
  prixSaisi: '42.50',
  creerBrouillon: 'Create the draft',
  etatAnnonce: 'Listing status',
  brouillon: 'Draft',
  enControle: 'Under review',
  publie: 'Published',
  refuse: 'Rejected',
  soumettre: 'Submit for review',
  avertissementRetrait: 'The listing will be removed from the catalogue',
  enregistrerEtRenvoyer: 'Save and send back for review',
  renvoyeAuControle: /revision 3: the listing is under review again/,
  motifIncomplet: 'Reason: Incomplete listing',
  erreurMontant: 'Invalid value. Amount example: 12.50.',
  erreurFormulaire: 'The form contains 1 error: fix the highlighted field.',
  accesReserve: 'Restricted access',
  echecConnexion: 'Sign-in did not complete',
  echecDemarrage: 'The application could not start',
  pageIntrouvable: 'Page not found',
  articleIntrouvable: 'Item not found',
  categorieInconnue: 'This category does not exist',
  articlesAVerifier: 'Items to review',
  publier: 'Publish',
  refuser: 'Reject',
  motifDuRefus: 'Rejection reason',
  motifManquant: 'Choose a rejection reason.',
  confirmerRefus: 'Confirm the rejection',
  motifPrixAbusif: 'Excessive price',
  motifControlePrix: 'Review reason: Price to be confirmed',
  articlePublie: /has been published\./,
  articleRefuse: /has been rejected\./,
  conflitAdmin: 'The item has changed, reload the list',
};

export const textes: Readonly<Record<Langue, Textes>> = { fr, en };
