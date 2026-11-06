import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { buyer, sellerA, signInAs } from '../US-008/identityProviderHarness';
import { axeTimeout, blockingViolations } from '../US-013/sellerMarketplaceHarness';
import {
  answerWith,
  expectCardCount,
  failureTimeout,
  givenNotifications,
  notification,
  notificationsScreenPath,
  openNotificationsAs,
  wording,
} from './sellerNotificationsHarness';

const languages: readonly SupportedLanguage[] = ['fr', 'en'];

type Situation = {
  readonly name: string;
  readonly open: (language: SupportedLanguage) => Promise<void>;
};

const situations: readonly Situation[] = [
  {
    name: 'liste des trois issues',
    open: async (language) => {
      givenNotifications(
        notification(3, { issue: 'refuse', motif: 'annonce_incomplete' }),
        notification(2, { issue: 'a_verifier', motif: 'categorie_inconnue' }),
        notification(1),
      );
      await openNotificationsAs(sellerA, language);
      await expectCardCount(3, language);
    },
  },
  {
    name: 'état vide',
    open: async (language) => {
      await openNotificationsAs(sellerA, language);
      await screen.findByRole('heading', { name: wording[language].empty });
    },
  },
  {
    name: 'état d’erreur',
    open: async (language) => {
      answerWith(503);
      await openNotificationsAs(sellerA, language);
      await screen.findByRole('heading', { name: wording[language].error }, failureTimeout);
    },
  },
  {
    name: 'session expirée (401)',
    open: async (language) => {
      answerWith(401);
      await openNotificationsAs(sellerA, language);
      await screen.findByRole('heading', { name: wording[language].expired });
    },
  },
  {
    name: 'accès réservé (403)',
    open: async (language) => {
      answerWith(403);
      await openNotificationsAs(sellerA, language);
      await screen.findByRole('heading', { name: wording[language].forbidden });
    },
  },
  {
    name: 'accès réservé (compte sans rôle vendeur)',
    open: async (language) => {
      await signInAs(buyer, notificationsScreenPath, language);
      await screen.findByRole('heading', { level: 1, name: wording[language].forbidden });
    },
  },
];

const cases = languages.flatMap((language) =>
  situations.map((situation) => [situation.name, language, situation] as const),
);

describe('US-028-AC10 axe-core sur la liste des notifications', () => {
  it.each(cases)(
    'Given l’écran des notifications (%s, %s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (_, language, situation) => {
      await situation.open(language);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it('Given la liste affichée, When elle est parcourue par rôles, Then elle expose un titre de niveau 1, une région nommée, une liste et un titre de niveau 3 par carte', async () => {
    givenNotifications(notification(2, { issue: 'refuse', motif: 'prix_abusif' }), notification(1));

    await openNotificationsAs();
    await expectCardCount(2);

    expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1);
    expect(screen.getByRole('region', { name: wording.fr.listTitle })).toBeInTheDocument();
    expect(screen.getAllByRole('heading', { level: 3 })).toHaveLength(2);
  });
});
