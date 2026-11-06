import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { administrator, buyer, sellerA, signInAs } from '../US-008/identityProviderHarness';
import {
  answerNormally,
  answerWith,
  expectCardCount,
  failureTimeout,
  givenNotifications,
  holdAnswers,
  notification,
  notificationReads,
  notificationsScreenPath,
  openNotificationsAs,
  wording,
} from './sellerNotificationsHarness';

const languages: readonly SupportedLanguage[] = ['fr', 'en'];

describe('US-028-AC6 côté front : réponse 401 de la liste', () => {
  it.each(languages)(
    'Given la langue %s et une liste qui répond 401, When le vendeur ouvre l’écran, Then l’état « session expirée » propose de se reconnecter',
    async (language) => {
      givenNotifications(notification(1));
      answerWith(401);

      await openNotificationsAs(sellerA, language);

      expect(
        await screen.findByRole('heading', { name: wording[language].expired }),
      ).toBeInTheDocument();
      expect(
        screen.getByRole('button', { name: wording[language].signInAgain }),
      ).toBeInTheDocument();
      expect(screen.queryAllByRole('listitem')).toHaveLength(0);
    },
  );

  it('Given une liste déjà affichée, When la relecture répond 401, Then l’état « session expirée » remplace la liste', async () => {
    givenNotifications(notification(1));
    await openNotificationsAs();
    await expectCardCount(1);

    answerWith(401);

    expect(
      await screen.findByRole('heading', { name: wording.fr.expired }, { timeout: 12000 }),
    ).toBeInTheDocument();
  });
});

describe('US-028-AC6 côté front : réponse 403 de la liste', () => {
  it.each(languages)(
    'Given la langue %s et une liste qui répond 403, When le vendeur ouvre l’écran, Then l’état « accès réservé » est affiché sans notification',
    async (language) => {
      givenNotifications(notification(1));
      answerWith(403);

      await openNotificationsAs(sellerA, language);

      expect(
        await screen.findByRole('heading', { name: wording[language].forbidden }),
      ).toBeInTheDocument();
      expect(screen.queryAllByRole('listitem')).toHaveLength(0);
      expect(screen.queryByText(wording[language].issues.publie)).not.toBeInTheDocument();
    },
  );

  it.each([
    ['acheteur', buyer],
    ['administrateur', administrator],
  ] as const)(
    'Given un compte %s sans rôle vendeur, When il ouvre l’écran des notifications, Then « Accès réservé » est affiché sans notification',
    async (_, account) => {
      givenNotifications(notification(1));

      await signInAs(account, notificationsScreenPath);

      expect(
        await screen.findByRole('heading', { level: 1, name: wording.fr.forbidden }),
      ).toBeInTheDocument();
      expect(screen.queryByText(wording.fr.issues.publie)).not.toBeInTheDocument();
    },
  );
});

describe('US-028 états de chargement et d’erreur', () => {
  it('Given une réponse qui tarde, When le vendeur ouvre l’écran, Then le chargement est annoncé puis la liste apparaît', async () => {
    givenNotifications(notification(1));
    const release = holdAnswers();

    await openNotificationsAs();

    expect(await screen.findByText(wording.fr.loading)).toBeInTheDocument();
    release();
    await expectCardCount(1);
    expect(screen.queryByText(wording.fr.loading)).not.toBeInTheDocument();
  });

  it.each(languages)(
    'Given la langue %s et un service indisponible, When le vendeur ouvre l’écran, Then un état d’erreur propose de réessayer',
    async (language) => {
      answerWith(503);

      await openNotificationsAs(sellerA, language);

      expect(
        await screen.findByRole('heading', { name: wording[language].error }, failureTimeout),
      ).toBeInTheDocument();
      expect(screen.getByRole('button', { name: wording[language].retry })).toBeInTheDocument();
    },
  );

  it('Given l’état d’erreur, When le service revient et que le vendeur réessaie, Then la liste s’affiche', async () => {
    givenNotifications(notification(1, { issue: 'refuse', motif: 'prix_abusif' }));
    answerWith(500);
    await openNotificationsAs();
    const retry = await screen.findByRole('button', { name: wording.fr.retry }, failureTimeout);

    answerNormally();
    await userEvent.click(retry);

    await expectCardCount(1);
    expect(screen.queryByRole('heading', { name: wording.fr.error })).not.toBeInTheDocument();
  });

  it('Given une réponse 200 au format invalide, When le vendeur ouvre l’écran, Then l’état d’erreur est affiché', async () => {
    answerWith(200, { elements: 'pas une liste', page: 1, taille: 20, total: 1 });

    await openNotificationsAs();

    expect(
      await screen.findByRole('heading', { name: wording.fr.error }, failureTimeout),
    ).toBeInTheDocument();
    expect(notificationReads().length).toBeGreaterThan(0);
  });
});
