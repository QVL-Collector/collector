import { within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { sellerA, switchLanguage } from '../US-008/identityProviderHarness';
import {
  cards,
  expectCardCount,
  givenNotifications,
  notification,
  openNotificationsAs,
  wording,
  type Issue,
  type ReasonCode,
  waitUntilSettled,
} from './sellerNotificationsHarness';

const languages: readonly SupportedLanguage[] = ['fr', 'en'];
const sellerTitle = 'Titre saisi par le vendeur : Montre Lip 1972';
const sellerDescription = 'Description saisie : boîtier rayé, appelez le 06 12 34 56 78';
const markup = '<img src=x onerror="window.__us028 = true"><b>Gras</b>';

const decisions: readonly (readonly [Issue, ReasonCode | null])[] = [
  ['publie', null],
  ['refuse', 'annonce_incomplete'],
  ['a_verifier', 'categorie_inconnue'],
];

const reasonsByLanguage = languages.flatMap((language) =>
  (Object.keys(wording[language].reasons) as ReasonCode[]).map(
    (reason) => [language, reason] as const,
  ),
);

const decisionsByLanguage = languages.flatMap((language) =>
  decisions.map(([issue, reason]) => [language, issue, reason] as const),
);

function sellerArticle(id: string) {
  return {
    id,
    etat: 'refuse',
    revision: 2,
    titre: sellerTitle,
    description: sellerDescription,
    categorie: { id: 'montres', libelle: 'Montres' },
    prix: { montantCentimes: 12000, devise: 'EUR' },
    fraisDePort: null,
    motif: 'annonce_incomplete',
    creeLe: '2026-11-01T08:00:00Z',
    modifieLe: '2026-11-01T08:00:00Z',
  };
}

function hrefOf(input: RequestInfo | URL): string {
  return typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
}

beforeEach(() => {
  const upstream = globalThis.fetch;
  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const { pathname } = new URL(hrefOf(input), window.location.origin);
      if (!pathname.startsWith('/marche/')) {
        return upstream(input, init);
      }
      const id = pathname.split('/').at(-1) ?? '';
      return new Response(JSON.stringify(sellerArticle(id)), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      });
    }),
  );
});

function onlyCard(language: SupportedLanguage = 'fr'): HTMLElement {
  const [card] = cards(language);
  if (card === undefined) {
    throw new Error('Aucune carte de notification');
  }
  return card;
}

function pageText(): string {
  return document.body.textContent;
}

describe('US-028-AC4 texte construit à partir de l’issue et de la catégorie de motif', () => {
  it.each(decisionsByLanguage)(
    'Given la langue %s et une décision %s (motif %s), When la notification est affichée, Then son titre et son motif sont ceux de la liste fermée',
    async (language, issue, reason) => {
      givenNotifications(notification(1, { issue, motif: reason }));

      await openNotificationsAs(sellerA, language);
      await expectCardCount(1, language);

      const card = within(cards(language)[0] ?? document.body);
      expect(card.getByRole('heading', { level: 3 })).toHaveTextContent(
        wording[language].issues[issue],
      );
      if (reason !== null) {
        expect(
          card.getByText(wording[language].reasons[reason], { exact: false }),
        ).toBeInTheDocument();
      }
    },
  );

  it.each(reasonsByLanguage)(
    'Given la langue %s et un refus pour le motif %s, When la notification est affichée, Then le motif est traduit et le code brut n’apparaît pas',
    async (language, reason) => {
      givenNotifications(notification(1, { issue: 'refuse', motif: reason }));

      await openNotificationsAs(sellerA, language);
      await expectCardCount(1, language);

      const card = cards(language)[0] ?? document.body;
      expect(card).toHaveTextContent(wording[language].reasons[reason]);
      expect(card.textContent).not.toContain(reason);
    },
  );

  it('Given une décision publiée, When la notification est affichée, Then aucun motif n’est mentionné', async () => {
    givenNotifications(notification(1, { issue: 'publie', motif: null }));

    await openNotificationsAs();
    await expectCardCount(1);

    const text = onlyCard().textContent;
    for (const label of Object.values(wording.fr.reasons)) {
      expect(text).not.toContain(label);
    }
  });

  it('Given des notifications affichées en français, When le vendeur passe en anglais, Then leur texte est reconstruit en anglais', async () => {
    givenNotifications(notification(1, { issue: 'refuse', motif: 'prix_abusif' }));
    await openNotificationsAs();
    await expectCardCount(1);

    await switchLanguage('en');

    await vi.waitFor(() => {
      expect(onlyCard('en')).toHaveTextContent(wording.en.issues.refuse);
    });
    expect(onlyCard('en')).toHaveTextContent(wording.en.reasons.prix_abusif);
    expect(onlyCard('en').textContent).not.toContain(wording.fr.issues.refuse);
    expect(onlyCard('en').textContent).not.toContain(wording.fr.reasons.prix_abusif);
  });
});

describe('US-028-AC4 aucun contenu saisi par un utilisateur', () => {
  it('Given un article du vendeur dont le titre et la description sont saisis, When sa notification est affichée, Then ni le titre ni la description n’apparaissent', async () => {
    givenNotifications(notification(1, { issue: 'refuse', motif: 'annonce_incomplete' }));

    await openNotificationsAs();
    await expectCardCount(1);

    expect(pageText()).not.toContain(sellerTitle);
    expect(pageText()).not.toContain(sellerDescription);
    expect(pageText()).not.toContain('Montre Lip');
  });

  it('Given une réponse qui porte en plus des champs libres (titre, commentaire, balisage), When l’écran s’affiche, Then aucun de ces contenus n’est rendu ni interprété', async () => {
    givenNotifications({
      ...notification(1, { issue: 'refuse', motif: 'prix_abusif' }),
      titre: sellerTitle,
      commentaire: markup,
      description: sellerDescription,
    });

    await openNotificationsAs();
    await waitUntilSettled();

    expect(pageText()).not.toContain(sellerTitle);
    expect(pageText()).not.toContain(sellerDescription);
    expect(pageText()).not.toContain('Gras');
    expect(document.querySelector('img[src="x"]')).toBeNull();
    expect('__us028' in window).toBe(false);
  });

  it('Given une réponse dont le motif est hors liste fermée, When l’écran s’affiche, Then le code reçu n’est jamais affiché', async () => {
    givenNotifications(
      notification(1, { issue: 'refuse', motif: 'motif_libre_saisi' as ReasonCode }),
    );

    await openNotificationsAs();
    await waitUntilSettled();

    expect(pageText()).not.toContain('motif_libre_saisi');
  });

  it('Given une réponse dont l’issue est hors liste fermée, When l’écran s’affiche, Then la valeur reçue n’est jamais affichée', async () => {
    givenNotifications(notification(1, { issue: 'Texte libre du vendeur' as Issue }));

    await openNotificationsAs();
    await waitUntilSettled();

    expect(pageText()).not.toContain('Texte libre du vendeur');
  });
});
