import type { ControlReason } from '../../../shared/controlReason';

type DemoAmount = {
  readonly montantCentimes: number;
  readonly devise: 'EUR';
};

export type DemoReviewArticle = {
  readonly id: string;
  readonly revision: number;
  readonly titre: string;
  readonly description: string;
  readonly categorie: { readonly id: string; readonly libelle: string } | null;
  readonly prix: DemoAmount | null;
  readonly fraisDePort: DemoAmount | null;
  readonly motif: ControlReason;
  readonly vendeurId: string;
  readonly modifieLe: string;
};

const centsPerEuro = 100;

function euros(amount: number): DemoAmount {
  return { montantCentimes: amount * centsPerEuro, devise: 'EUR' };
}

export const demoReviewArticleChangedBySellerId = 'e4b1c8d2-6a3f-4d9e-8b17-2c5f9a0d3e11';

export const demoReviewArticles: readonly DemoReviewArticle[] = [
  {
    id: '3f8a2b6c-1d4e-4a7b-9c0d-5e6f7a8b9c21',
    revision: 2,
    titre: '<script>alert(1)</script> Carte rare holographique',
    description: 'Carte en excellent état, protégée sous pochette depuis l’achat.',
    categorie: { id: 'cartes', libelle: 'Cartes à collectionner' },
    prix: euros(4800),
    fraisDePort: euros(6),
    motif: 'prix_a_confirmer',
    vendeurId: '7d2e9f14-3b5a-4c68-a1e0-9f8b7c6d5e41',
    modifieLe: '2026-10-26T15:20:00Z',
  },
  {
    id: 'a9c3e5f7-2b4d-4e6f-8a1c-3d5e7f9b1c31',
    revision: 1,
    titre: '<b>gras</b> Figurine de collection numérotée',
    description: 'Édition limitée, boîte d’origine, certificat d’authenticité inclus.',
    categorie: { id: 'figurines', libelle: 'Figurines' },
    prix: euros(210),
    fraisDePort: null,
    motif: 'vendeur_a_verifier',
    vendeurId: '2a4c6e8f-1b3d-4f5a-9c7e-0d2f4a6c8e51',
    modifieLe: '2026-10-25T10:05:00Z',
  },
  {
    id: demoReviewArticleChangedBySellerId,
    revision: 3,
    titre: 'Pièce de 5 francs Semeuse, 1960',
    description: 'Argent, belle patine, quelques micro-rayures visibles à la loupe.',
    categorie: { id: 'monnaies', libelle: 'Monnaies et billets' },
    prix: euros(35),
    fraisDePort: euros(3),
    motif: 'prix_a_confirmer',
    vendeurId: '7d2e9f14-3b5a-4c68-a1e0-9f8b7c6d5e41',
    modifieLe: '2026-10-24T08:45:00Z',
  },
];
