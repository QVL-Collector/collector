import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { buyer, openCollector, press, signInAs } from '../US-008/identityProviderHarness';
import {
  articleId,
  expectNoActionableReservation,
  expectNoReservationSent,
  failureTimeout,
  givenArticles,
  givenNextOrder,
  openArticleAs,
  order,
  publishedArticle,
  wording,
} from './reservationHarness';

const languages: readonly SupportedLanguage[] = ['fr', 'en'];

const reservedTitle = 'Figurine déjà réservée';
const freeTitle = 'Figurine encore libre';

function givenOneReservedAndOneFree(): void {
  givenArticles(
    publishedArticle(1, { titre: reservedTitle, reserve: true }),
    publishedArticle(2, { titre: freeTitle, reserve: false }),
  );
}

async function cardOf(title: string): Promise<HTMLElement> {
  const heading = await screen.findByText(title, {}, failureTimeout);
  const card = heading.closest('li');
  if (card === null) {
    throw new Error(`Carte de « ${title} » absente`);
  }
  return card;
}

describe('US-030-AC3 : l’article réservé porte la mention « réservé » et n’est plus réservable', () => {
  it.each(languages)(
    'Given la langue %s, un article réservé et un libre au catalogue, When un visiteur consulte le catalogue, Then seule la carte réservée porte la mention',
    async (language) => {
      givenOneReservedAndOneFree();
      await openCollector('/', language);

      expect(
        within(await cardOf(reservedTitle)).getByText(wording[language].reserved),
      ).toBeInTheDocument();
      expect(
        within(await cardOf(freeTitle)).queryByText(wording[language].reserved),
      ).not.toBeInTheDocument();
    },
  );

  it.each(languages)(
    'Given la langue %s et un article réservé, When un visiteur ouvre la fiche, Then la mention « réservé » et le motif sont affichés, sans offre de réservation actionnable',
    async (language) => {
      givenOneReservedAndOneFree();
      await openArticleAs(null, articleId(1), language);

      expect(
        await screen.findByText(wording[language].reserved, {}, failureTimeout),
      ).toBeInTheDocument();
      expect(screen.getByText(wording[language].reservedReason)).toBeInTheDocument();
      expectNoActionableReservation(language);
    },
  );

  it.each(languages)(
    'Given la langue %s et un article réservé, When un acheteur connecté ouvre la fiche, Then aucun bouton « Réserver » actionnable n’est proposé et rien n’est envoyé',
    async (language) => {
      givenOneReservedAndOneFree();
      await openArticleAs(buyer, articleId(1), language);

      expect(
        await screen.findByText(wording[language].reservedReason, {}, failureTimeout),
      ).toBeInTheDocument();
      expectNoActionableReservation(language);
      await expectNoReservationSent();
    },
  );

  it('Given un article libre, When un visiteur ouvre la fiche, Then aucune mention « réservé » n’est affichée', async () => {
    givenOneReservedAndOneFree();
    await openArticleAs(null, articleId(2));

    expect(
      await screen.findByText(wording.fr.signInPrompt, {}, failureTimeout),
    ).toBeInTheDocument();
    expect(screen.queryByText(wording.fr.reserved)).not.toBeInTheDocument();
  });

  it('Given un acheteur qui vient de réserver, When il revient au catalogue, Then l’article porte la mention « réservé »', async () => {
    givenArticles(publishedArticle(2, { titre: freeTitle }));
    givenNextOrder(
      order(1, articleId(2), {
        prix: 12000,
        frais: 590,
        total: 12590,
        commission: 630,
        vendeur: 11960,
      }),
    );
    await signInAs(buyer, `/articles/${articleId(2)}`);
    await press(await screen.findByRole('button', { name: wording.fr.reserve }, failureTimeout));
    await screen.findByRole('heading', { level: 1, name: wording.fr.title }, failureTimeout);

    await press(screen.getByRole('link', { name: 'Retour au catalogue' }));

    expect(within(await cardOf(freeTitle)).getByText(wording.fr.reserved)).toBeInTheDocument();
  });

  it('Given un acheteur qui vient de réserver, When il rouvre la fiche par « Voir l’article », Then elle porte la mention et n’est plus réservable', async () => {
    givenArticles(publishedArticle(2, { titre: freeTitle }));
    givenNextOrder(
      order(1, articleId(2), {
        prix: 12000,
        frais: 590,
        total: 12590,
        commission: 630,
        vendeur: 11960,
      }),
    );
    await signInAs(buyer, `/articles/${articleId(2)}`);
    await press(await screen.findByRole('button', { name: wording.fr.reserve }, failureTimeout));
    await screen.findByRole('heading', { level: 1, name: wording.fr.title }, failureTimeout);

    await press(screen.getByRole('link', { name: 'Voir l’article' }));

    expect(
      await screen.findByText(wording.fr.reservedReason, {}, failureTimeout),
    ).toBeInTheDocument();
    expectNoActionableReservation('fr');
  });
});
