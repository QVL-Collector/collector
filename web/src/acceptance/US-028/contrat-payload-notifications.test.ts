import { describe, expect, it } from 'vitest';
import { parseNotificationPage } from '../../features/notifications/api/notificationPayload';
import { notification, type NotificationWire } from './notificationFixtures';

function page(elements: unknown): unknown {
  return {
    elements,
    page: 1,
    taille: 20,
    total: Array.isArray(elements) ? elements.length : 0,
  };
}

function withField(field: keyof NotificationWire, value: unknown): unknown {
  return { ...notification(1, { issue: 'refuse', motif: 'prix_abusif' }), [field]: value };
}

function without(field: keyof NotificationWire): unknown {
  const element: Record<string, unknown> = { ...notification(1) };
  Reflect.deleteProperty(element, field);
  return element;
}

describe('US-028 contrat de la réponse GET /interactions/v1/notifications', () => {
  it('Given une page valide avec les trois issues, When elle est lue, Then chaque notification garde son identité, son issue, son motif et sa date', () => {
    const elements = [
      notification(3, { issue: 'refuse', motif: 'vendeur_non_fiable', revision: 4 }),
      notification(2, { issue: 'a_verifier', motif: 'prix_a_confirmer' }),
      notification(1, { issue: 'publie', motif: null }),
    ];

    const parsed = parseNotificationPage(page(elements));

    expect(parsed).toHaveLength(3);
    expect(
      parsed.map((item) => [item.id, item.articleId, item.revision, item.issue, item.reason]),
    ).toEqual(
      elements.map((item) => [item.id, item.articleId, item.revision, item.issue, item.motif]),
    );
    expect(parsed.map((item) => item.decidedAt.toISOString())).toEqual(
      elements.map((item) => item.decideLe),
    );
  });

  it('Given une page vide, When elle est lue, Then aucune notification n’est rendue', () => {
    expect(parseNotificationPage(page([]))).toEqual([]);
  });

  it('Given une date au format microsecondes, When elle est lue, Then elle est interprétée', () => {
    const parsed = parseNotificationPage(
      page([notification(1, { decideLe: '2026-11-04T09:42:00.000000Z' })]),
    );

    expect(parsed[0]?.decidedAt.toISOString()).toBe('2026-11-04T09:42:00.000Z');
  });

  it.each([
    ['null', null],
    ['une chaîne', 'notifications'],
    ['un tableau nu', [notification(1)]],
    ['sans elements', { page: 1, taille: 20, total: 0 }],
    ['elements qui n’est pas une liste', page({ id: 'x' })],
  ])('Given une réponse %s, When elle est lue, Then elle est rejetée', (_, payload) => {
    expect(() => parseNotificationPage(payload)).toThrow();
  });

  it.each([
    ['issue hors liste fermée', withField('issue', 'supprime')],
    ['issue libre', withField('issue', 'Texte libre du vendeur')],
    ['motif hors liste fermée', withField('motif', 'motif_inconnu')],
    ['motif libre', withField('motif', 'Votre annonce est nulle')],
    ['motif numérique', withField('motif', 3)],
    ['date illisible', withField('decideLe', 'hier')],
    ['révision textuelle', withField('revision', 'deux')],
    ['id absent', without('id')],
    ['articleId absent', without('articleId')],
    ['issue absente', without('issue')],
    ['date absente', without('decideLe')],
  ])('Given un élément avec %s, When la page est lue, Then elle est rejetée', (_, element) => {
    expect(() => parseNotificationPage(page([notification(2), element]))).toThrow();
  });

  it.each([
    ['motif absent sur une décision publiée', without('motif')],
    ['motif absent sur un refus', withoutMotif({ issue: 'refuse', motif: 'prix_abusif' })],
    [
      'motif absent sur un article à vérifier',
      withoutMotif({ issue: 'a_verifier', motif: 'prix_a_confirmer' }),
    ],
    ['motif sur une décision publiée', notification(1, { issue: 'publie', motif: 'prix_abusif' })],
    ['motif null sur un refus', notification(1, { issue: 'refuse', motif: null })],
    ['motif null sur un article à vérifier', notification(1, { issue: 'a_verifier', motif: null })],
  ])('Given un élément avec %s, When la page est lue, Then elle est rejetée', (_, element) => {
    expect(() => parseNotificationPage(page([notification(2), element]))).toThrow();
  });

  it.each([
    ['publié sans motif', notification(1, { issue: 'publie', motif: null })],
    ['refusé avec motif', notification(1, { issue: 'refuse', motif: 'article_interdit' })],
    [
      'à vérifier avec motif',
      notification(1, { issue: 'a_verifier', motif: 'vendeur_a_verifier' }),
    ],
  ])('Given un élément %s, When la page est lue, Then il est accepté', (_, element) => {
    expect(parseNotificationPage(page([element]))).toHaveLength(1);
  });
});

function withoutMotif(overrides: Partial<NotificationWire>): unknown {
  const element: Record<string, unknown> = { ...notification(1, overrides) };
  Reflect.deleteProperty(element, 'motif');
  return element;
}
