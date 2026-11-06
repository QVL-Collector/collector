import { controlReasons, type ControlReason } from './controlReason';

export const refusalReasons = [
  'annonce_incomplete',
  'coordonnees_detectees',
  'prix_abusif',
  'article_interdit',
  'vendeur_non_fiable',
] as const;

export type RefusalReason = (typeof refusalReasons)[number];

export type ArticleReason = ControlReason | RefusalReason;

export const articleReasons: readonly ArticleReason[] = [
  ...new Set<ArticleReason>([...controlReasons, ...refusalReasons]),
];
