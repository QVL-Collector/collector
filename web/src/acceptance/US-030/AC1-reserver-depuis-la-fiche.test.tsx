import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import {
  buyer,
  currentPath,
  identityProvider,
  press,
  sellerA,
} from '../US-008/identityProviderHarness';
import {
  articleId,
  confirmationPath,
  expectNoReservationSent,
  failureTimeout,
  givenArticles,
  givenNextOrder,
  givenOrderAlreadyCreated,
  openArticleAs,
  order,
  orderId,
  publishedArticle,
  reservationRequests,
  waitForReservationRequest,
  wording,
} from './reservationHarness';

const languages: readonly SupportedLanguage[] = ['fr', 'en'];

const standardAmounts = { prix: 12000, frais: 590, total: 12590, commission: 630, vendeur: 11960 };

function givenReservableArticle(): void {
  givenArticles(publishedArticle(1));
  givenNextOrder(order(1, articleId(1), standardAmounts));
}

describe('US-030-AC1 : un acheteur connecté réserve un article publié disponible', () => {
  it.each(languages)(
    'Given la langue %s, un acheteur connecté sur la fiche d’un article disponible, When il clique « Réserver », Then la commande est demandée à Transactions avec son jeton et l’écran de confirmation s’affiche',
    async (language) => {
      givenReservableArticle();
      await openArticleAs(buyer, articleId(1), language);

      await press(
        await screen.findByRole('button', { name: wording[language].reserve }, failureTimeout),
      );

      const sent = await waitForReservationRequest();
      expect(sent.authorization).toBe(
        `Bearer ${identityProvider().issuedAccessTokens.at(-1) ?? ''}`,
      );
      expect(
        await screen.findByRole(
          'heading',
          { level: 1, name: wording[language].title },
          failureTimeout,
        ),
      ).toBeInTheDocument();
      expect(currentPath()).toBe(confirmationPath(orderId(1)));
      expect(screen.getByText(wording[language].status)).toBeInTheDocument();
    },
  );

  it('Given un acheteur connecté, When il clique une seule fois « Réserver », Then une seule demande de commande part vers Transactions', async () => {
    givenReservableArticle();
    await openArticleAs(buyer, articleId(1));

    await press(await screen.findByRole('button', { name: wording.fr.reserve }, failureTimeout));

    await screen.findByRole('heading', { level: 1, name: wording.fr.title }, failureTimeout);
    expect(reservationRequests()).toHaveLength(1);
  });

  it.each(languages)(
    'Given la langue %s et un visiteur non connecté sur la fiche, When il consulte l’article, Then il est invité à se connecter et aucune réservation n’est envoyée',
    async (language) => {
      givenReservableArticle();
      await openArticleAs(null, articleId(1), language);

      expect(
        await screen.findByText(wording[language].signInPrompt, {}, failureTimeout),
      ).toBeInTheDocument();
      expect(screen.getByRole('button', { name: wording[language].signIn })).toBeInTheDocument();
      expect(
        screen.queryByRole('button', { name: wording[language].reserve }),
      ).not.toBeInTheDocument();
      await expectNoReservationSent();
    },
  );

  it('Given un compte sans le rôle acheteur sur la fiche, When il consulte l’article, Then la réservation lui est expliquée comme réservée aux acheteurs et rien n’est envoyé', async () => {
    givenReservableArticle();
    await openArticleAs(sellerA, articleId(1));

    expect(await screen.findByText(wording.fr.buyerOnly, {}, failureTimeout)).toBeInTheDocument();
    expect(screen.queryByRole('button', { name: wording.fr.reserve })).not.toBeInTheDocument();
    await expectNoReservationSent();
  });

  it('Given un acheteur connecté, When Transactions répond que la commande existe déjà (200), Then l’écran de confirmation de cette commande s’affiche', async () => {
    givenReservableArticle();
    givenOrderAlreadyCreated();
    await openArticleAs(buyer, articleId(1));

    await press(await screen.findByRole('button', { name: wording.fr.reserve }, failureTimeout));

    await waitFor(() => {
      expect(currentPath()).toBe(confirmationPath(orderId(1)));
    }, failureTimeout);
  });
});
