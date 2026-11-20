import { act, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { listRequests } from '../US-030/reservationHarness';
import {
  buyerSpaceWording,
  expectState,
  givenBuyerOrders,
  itemFor,
  languages,
  openBuyerSpace,
  pendingOrder,
  replaceOrder,
  reservationItems,
} from './buyerSpaceHarness';

const watch = 'Montre de gousset 1920';
const pollingPeriodMs = 30_000;
const refreshMargin = { timeout: pollingPeriodMs + 10_000, interval: 250 };
const longScenario = 90_000;

async function expectExpiredAfterRefresh(language: 'fr' | 'en'): Promise<void> {
  const wording = buyerSpaceWording[language];
  await waitFor(async () => {
    expectState(
      itemFor(await reservationItems(language), watch),
      wording.expired,
      wording.awaitingPayment,
    );
  }, refreshMargin);
}

describe('US-032-AC2 : une réservation dont l’échéance passe pendant la consultation devient « expirée » au rafraîchissement', () => {
  it.each(languages)(
    'Given la langue %s et une réservation en attente affichée, When l’échéance passe et que la liste est rafraîchie, Then son état devient « expirée »',
    async (language) => {
      const deadline = new Date(Date.now() + 4_000).toISOString();
      const pending = pendingOrder(1, watch, deadline);
      givenBuyerOrders(pending);
      await openBuyerSpace(language);
      const wording = buyerSpaceWording[language];
      expectState(
        itemFor(await reservationItems(language), watch),
        wording.awaitingPayment,
        wording.expired,
      );

      const expiry = setTimeout(
        () => {
          replaceOrder({ ...pending, statut: 'expiree' });
        },
        new Date(deadline).getTime() - Date.now(),
      );
      const requestsBefore = listRequests().length;

      try {
        await expectExpiredAfterRefresh(language);
      } finally {
        clearTimeout(expiry);
      }
      expect(listRequests().length).toBeGreaterThan(requestsBefore);
    },
    longScenario,
  );

  it(
    'Given une réservation en attente affichée et une échéance encore lointaine, When le serveur la renvoie expirée au prochain rafraîchissement périodique, Then l’écran la montre « expirée » sans action de l’acheteur',
    async () => {
      const pending = pendingOrder(1, watch);
      givenBuyerOrders(pending);
      await openBuyerSpace('fr');
      await reservationItems('fr');
      const startedAt = Date.now();

      replaceOrder({ ...pending, statut: 'expiree' });

      await expectExpiredAfterRefresh('fr');
      expect(Date.now() - startedAt).toBeLessThanOrEqual(pollingPeriodMs + 10_000);
    },
    longScenario,
  );

  it('Given une réservation en attente affichée, When l’acheteur revient sur l’onglet après que le serveur l’a passée expirée, Then l’écran la montre « expirée »', async () => {
    const pending = pendingOrder(1, watch);
    givenBuyerOrders(pending);
    await openBuyerSpace('fr');
    await reservationItems('fr');
    replaceOrder({ ...pending, statut: 'expiree' });

    act(() => {
      window.dispatchEvent(new Event('focus'));
      window.dispatchEvent(new Event('visibilitychange'));
    });

    const wording = buyerSpaceWording.fr;
    await waitFor(
      async () => {
        expectState(
          itemFor(await reservationItems('fr'), watch),
          wording.expired,
          wording.awaitingPayment,
        );
      },
      { timeout: 5_000 },
    );
  });

  it('Given une réservation expirée, When la liste est rafraîchie, Then elle reste « expirée » et son article, son montant et son échéance restent affichés', async () => {
    const pending = pendingOrder(1, watch);
    givenBuyerOrders({ ...pending, statut: 'expiree' });
    await openBuyerSpace('fr');
    const item = itemFor(await reservationItems('fr'), watch);
    const before = item.textContent;

    act(() => {
      window.dispatchEvent(new Event('focus'));
      window.dispatchEvent(new Event('visibilitychange'));
    });
    await new Promise((resolve) => setTimeout(resolve, 500));

    const after = itemFor(await reservationItems('fr'), watch);
    expectState(after, buyerSpaceWording.fr.expired, buyerSpaceWording.fr.awaitingPayment);
    expect(within(after).getByText(watch)).toBeInTheDocument();
    expect(after.textContent).toBe(before);
  });
});
