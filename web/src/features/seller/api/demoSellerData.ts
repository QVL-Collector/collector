import type { ArticleReason } from '../../../shared/articleReason';
import type { ArticleState } from '../domain/sellerArticle';
import { toAmountPayload, type AmountPayload } from './draftPayload';

export type DemoCategory = {
  readonly id: string;
  readonly libelle: string;
};

type DemoAmount = AmountPayload | null;

export type DemoArticle = {
  readonly id: string;
  readonly etat: ArticleState;
  readonly revision: number;
  readonly titre: string;
  readonly description: string;
  readonly categorie: DemoCategory | null;
  readonly prix: DemoAmount;
  readonly fraisDePort: DemoAmount;
  readonly motif: ArticleReason | null;
  readonly creeLe: string;
  readonly modifieLe: string;
};

export const demoCategoryUnknownToControl = 'objets-divers';

export const demoCategories: readonly DemoCategory[] = [
  { id: 'cartes', libelle: 'Cartes à collectionner' },
  { id: 'figurines', libelle: 'Figurines' },
  { id: 'jeux-video', libelle: 'Jeux vidéo rétro' },
  { id: 'livres', libelle: 'Livres et bandes dessinées' },
  { id: 'monnaies', libelle: 'Monnaies et billets' },
  { id: demoCategoryUnknownToControl, libelle: 'Objets divers' },
];

function category(id: string): DemoCategory | null {
  return demoCategories.find((candidate) => candidate.id === id) ?? null;
}

const centsPerEuro = 100;

function euros(amount: number): DemoAmount {
  return toAmountPayload(amount * centsPerEuro);
}

export const demoArticleModifiedElsewhereId = '5d0c4f7e-8a43-4b8e-9d55-3f1c2a7b9e05';

export const demoSellerArticles: readonly DemoArticle[] = [
  {
    id: '1b7e2c9a-4f3d-4c11-8a6e-2d9b0f5c7a01',
    etat: 'brouillon',
    revision: 2,
    titre: 'Console portable de 1989 en boîte d’origine',
    description: 'Console complète avec notice et câble, écran sans rayure.',
    categorie: category('jeux-video'),
    prix: euros(180),
    fraisDePort: euros(9),
    motif: null,
    creeLe: '2026-10-20T09:12:00Z',
    modifieLe: '2026-10-24T17:40:00Z',
  },
  {
    id: '7f4a1d2e-9c8b-4e57-b3a0-6e2f8d1c4b02',
    etat: 'refuse',
    revision: 3,
    titre: 'Lot de douze figurines peintes à la main',
    description: '',
    categorie: category('figurines'),
    prix: euros(65),
    fraisDePort: euros(7),
    motif: 'annonce_incomplete',
    creeLe: '2026-10-18T14:05:00Z',
    modifieLe: '2026-10-23T08:22:00Z',
  },
  {
    id: 'c2e9b4a7-1d6f-4f80-9a3b-8c5e7d2f1a03',
    etat: 'publie',
    revision: 2,
    titre: 'Affiche de cinéma originale, 1962',
    description: 'Affiche entoilée, format 120 × 160, petite déchirure en marge.',
    categorie: category('livres'),
    prix: euros(420),
    fraisDePort: euros(25),
    motif: null,
    creeLe: '2026-10-10T10:00:00Z',
    modifieLe: '2026-10-21T12:30:00Z',
  },
  {
    id: '9a3d7c1e-5b2f-4a68-8e4d-1f7b6c9a2d04',
    etat: 'a_verifier',
    revision: 2,
    titre: 'Boîte à musique ancienne en noyer',
    description: 'Mécanisme d’origine, joue deux airs, clé fournie.',
    categorie: category(demoCategoryUnknownToControl),
    prix: euros(150),
    fraisDePort: euros(12),
    motif: 'categorie_inconnue',
    creeLe: '2026-10-15T16:45:00Z',
    modifieLe: '2026-10-22T09:10:00Z',
  },
  {
    id: demoArticleModifiedElsewhereId,
    etat: 'brouillon',
    revision: 1,
    titre: 'Album de timbres de France, 1950 à 1970',
    description: 'Album complet, timbres neufs avec charnière.',
    categorie: category('monnaies'),
    prix: euros(95),
    fraisDePort: null,
    motif: null,
    creeLe: '2026-10-25T11:00:00Z',
    modifieLe: '2026-10-25T11:00:00Z',
  },
];
