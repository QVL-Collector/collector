import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { press, sellerA, signInAs } from '../US-008/identityProviderHarness';
import {
  backends,
  confirmationPath,
  failureTimeout,
  formatted,
  listAnswers,
  listRequests,
  openOrderAs,
  orderId,
} from '../US-030/reservationHarness';
import {
  buyerSpacePath,
  buyerSpaceWording,
  deadlinesIn,
  expectState,
  expiredOrder,
  givenBuyerOrders,
  itemFor,
  languages,
  nominalAmounts,
  normalizedText,
  openBuyerSpace,
  pendingOrder,
  reservationItems,
} from './buyerSpaceHarness';

const watch = 'Montre de gousset 1920';
const card = 'Carte « rare » n°7';
const coin = 'Pièce de 5 francs Semeuse';

describe('US-032-AC1 : l’acheteur retrouve ses réservations avec article, montant total, échéance et état', () => {
  it.each(languages)(
    'Given la langue %s et un acheteur avec une réservation en attente et une expirée, When il ouvre son espace, Then il voit pour chacune l’article, le montant total, l’échéance et l’état',
    async (language) => {
      const pending = pendingOrder(1, watch);
      const expired = expiredOrder(2, card);
      givenBuyerOrders(pending, expired);

      await openBuyerSpace(language);
      const items = await reservationItems(language);

      expect(items).toHaveLength(2);
      const pendingItem = itemFor(items, watch);
      const expiredItem = itemFor(items, card);
      for (const [item, wire] of [
        [pendingItem, pending],
        [expiredItem, expired],
      ] as const) {
        expect(normalizedText(item)).toContain(formatted(nominalAmounts.total, language));
        expect(deadlinesIn(item)).toContain(new Date(wire.echeance).getTime());
      }
      const wording = buyerSpaceWording[language];
      expectState(pendingItem, wording.awaitingPayment, wording.expired);
      expectState(expiredItem, wording.expired, wording.awaitingPayment);
    },
  );

  it('Given trois réservations renvoyées de la plus récente à la plus ancienne, When l’acheteur ouvre son espace, Then elles sont présentées dans cet ordre', async () => {
    givenBuyerOrders(pendingOrder(3, coin), expiredOrder(2, card), expiredOrder(1, watch));

    await openBuyerSpace('fr');
    const items = await reservationItems('fr');

    expect(
      items.map((item) => [coin, card, watch].find((title) => within(item).queryByText(title))),
    ).toEqual([coin, card, watch]);
  });

  it('Given un acheteur connecté, When il ouvre son espace, Then la liste est demandée à Transactions avec son jeton', async () => {
    givenBuyerOrders(pendingOrder(1, watch));

    await openBuyerSpace('fr');
    await reservationItems('fr');

    const request = listRequests().at(0);
    expect(request?.authorization).toMatch(/^Bearer \S+/);
    expect(request?.query.has('acheteurId')).toBe(false);
  });

  it.each(languages)(
    'Given la langue %s et une réservation, When l’acheteur suit le lien de la réservation, Then il arrive sur son détail',
    async (language) => {
      givenBuyerOrders(pendingOrder(1, watch));

      await openBuyerSpace(language);
      const item = itemFor(await reservationItems(language), watch);
      const link = within(item).getByRole('link', {
        name: buyerSpaceWording[language].viewDetail(watch),
      });

      expect(link.getAttribute('href')).toBe(confirmationPath(orderId(1)));
      await press(link);
      await waitFor(() => {
        expect(window.location.pathname).toBe(confirmationPath(orderId(1)));
      }, failureTimeout);
    },
  );

  it.each(languages)(
    'Given la langue %s et un acheteur sans réservation, When il ouvre son espace, Then un état vide l’invite à parcourir le catalogue',
    async (language) => {
      givenBuyerOrders();

      await openBuyerSpace(language);

      const wording = buyerSpaceWording[language];
      expect(
        await screen.findByRole('heading', { name: wording.emptyTitle }, failureTimeout),
      ).toBeInTheDocument();
      expect(screen.getByRole('link', { name: wording.emptyAction }).getAttribute('href')).toBe(
        '/',
      );
    },
  );

  it.each(languages)(
    'Given la langue %s et Transactions indisponible, When l’acheteur ouvre son espace puis réessaie une fois le service revenu, Then l’erreur est annoncée puis la liste s’affiche',
    async (language) => {
      givenBuyerOrders(pendingOrder(1, watch));
      listAnswers(503);

      await openBuyerSpace(language);
      const wording = buyerSpaceWording[language];
      await screen.findByRole('heading', { name: wording.errorTitle }, failureTimeout);
      backends().listAnswer = null;
      await press(screen.getByRole('button', { name: wording.retry }));

      itemFor(await reservationItems(language), watch);
    },
  );

  it.each(languages)(
    'Given la langue %s et un acheteur connecté, When il utilise l’entrée de navigation « Mes réservations », Then il arrive sur son espace',
    async (language) => {
      givenBuyerOrders(pendingOrder(1, watch));
      await openOrderAs(orderId(9), language);

      const entry = await screen.findByRole(
        'link',
        { name: buyerSpaceWording[language].navigation },
        failureTimeout,
      );
      await press(entry);

      await screen.findByRole(
        'heading',
        { level: 1, name: buyerSpaceWording[language].title },
        failureTimeout,
      );
      expect(window.location.pathname).toBe(buyerSpacePath);
    },
  );

  it('Given un compte vendeur sans rôle acheteur, When il ouvre l’espace acheteur, Then l’accès lui est refusé', async () => {
    givenBuyerOrders(pendingOrder(1, watch));
    await signInAs(sellerA, buyerSpacePath, 'fr');

    expect(
      await screen.findByRole(
        'heading',
        { level: 1, name: buyerSpaceWording.fr.accessDenied },
        failureTimeout,
      ),
    ).toBeInTheDocument();
    expect(screen.queryByText(watch)).not.toBeInTheDocument();
  });

  it.each(languages)(
    'Given la langue %s et une commande créée, When l’acheteur consulte sa confirmation, Then le titre figé de l’article y figure',
    async (language) => {
      givenBuyerOrders(pendingOrder(1, watch));
      await openOrderAs(orderId(1), language);

      expect(await screen.findByText(watch, {}, failureTimeout)).toBeInTheDocument();
    },
  );
});
