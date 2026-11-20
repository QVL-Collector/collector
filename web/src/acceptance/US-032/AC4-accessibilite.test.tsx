import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { axeTimeout, blockingViolations } from '../US-013/sellerMarketplaceHarness';
import { failureTimeout, listAnswers } from '../US-030/reservationHarness';
import {
  buyerSpaceWording,
  expiredOrder,
  givenBuyerOrders,
  languages,
  openBuyerSpace,
  pendingOrder,
  reservationItems,
} from './buyerSpaceHarness';

describe('US-032-AC4 : la liste des réservations sans violation axe-core serious ou critical (jsdom)', () => {
  it.each(languages)(
    'Given la langue %s et une liste avec une réservation en attente et une expirée, When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      givenBuyerOrders(pendingOrder(1, 'Montre de gousset 1920'), expiredOrder(2, 'Carte rare'));
      await openBuyerSpace(language);
      await reservationItems(language);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(languages)(
    'Given la langue %s et un acheteur sans réservation, When axe-core analyse l’état vide, Then aucune violation serious ou critical',
    async (language) => {
      givenBuyerOrders();
      await openBuyerSpace(language);
      await screen.findByRole(
        'heading',
        { name: buyerSpaceWording[language].emptyTitle },
        failureTimeout,
      );

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(languages)(
    'Given la langue %s et Transactions indisponible, When axe-core analyse l’état d’erreur, Then aucune violation serious ou critical',
    async (language) => {
      listAnswers(503);
      await openBuyerSpace(language);
      await screen.findByRole(
        'heading',
        { name: buyerSpaceWording[language].errorTitle },
        failureTimeout,
      );

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );
});
