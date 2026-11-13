export type Issue = 'publie' | 'refuse' | 'a_verifier';

export type ReasonCode =
  | 'annonce_incomplete'
  | 'coordonnees_detectees'
  | 'compte_non_autorise'
  | 'prix_a_confirmer'
  | 'vendeur_a_verifier'
  | 'categorie_inconnue'
  | 'prix_abusif'
  | 'article_interdit'
  | 'vendeur_non_fiable';

export type NotificationWire = {
  readonly id: string;
  readonly articleId: string;
  readonly revision: number;
  readonly issue: Issue;
  readonly motif: ReasonCode | null;
  readonly decideLe: string;
};

export function notificationId(rank: number): string {
  return `00000000-0000-4000-9000-${rank.toString().padStart(12, '0')}`;
}

export function articleOf(rank: number): string {
  return `00000000-0000-4000-a000-${rank.toString().padStart(12, '0')}`;
}

export function notification(
  rank: number,
  overrides: Partial<NotificationWire> = {},
): NotificationWire {
  return {
    id: notificationId(rank),
    articleId: articleOf(rank),
    revision: 2,
    issue: 'publie',
    motif: null,
    decideLe: new Date(Date.UTC(2026, 10, 4, 9, rank)).toISOString(),
    ...overrides,
  };
}
