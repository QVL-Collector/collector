import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { sellerA, signInAs } from '../US-008/identityProviderHarness';
import {
  addNotification,
  announcements,
  cardHeadings,
  elapse,
  expectCardCount,
  givenNotifications,
  hideTab,
  maximumDelayMs,
  notification,
  notificationReads,
  notificationsScreenPath,
  openNotificationsAs,
  pollingIntervalMs,
  wording,
} from './sellerNotificationsHarness';

const fr = wording.fr;
const immediately = 300;

async function openWithFakeTimers(): Promise<void> {
  vi.useFakeTimers({
    shouldAdvanceTime: true,
    toFake: ['setTimeout', 'clearTimeout', 'setInterval', 'clearInterval'],
  });
  await openNotificationsAs();
}

describe('US-028-AC2 liste des notifications dans l’espace vendeur', () => {
  it('Given le vendeur connecté sur le catalogue, When il suit le lien « Notifications », Then il arrive sur l’écran des notifications', async () => {
    givenNotifications(notification(1));
    await signInAs(sellerA, '/');

    await userEvent.click(await screen.findByRole('link', { name: fr.link }));

    expect(await screen.findByRole('heading', { level: 1, name: fr.title })).toBeInTheDocument();
    expect(window.location.pathname).toBe(notificationsScreenPath);
  });

  it('Given trois décisions, When le vendeur ouvre l’écran, Then la région « Décisions sur vos articles » les liste toutes dans l’ordre reçu', async () => {
    givenNotifications(
      notification(3, { issue: 'refuse', motif: 'prix_abusif' }),
      notification(2, { issue: 'a_verifier', motif: 'categorie_inconnue' }),
      notification(1),
    );

    await openNotificationsAs();

    await expectCardCount(3);
    expect(cardHeadings()).toEqual([fr.issues.refuse, fr.issues.a_verifier, fr.issues.publie]);
    expect(announcements()).toContain(fr.loaded(3));
  });

  it('Given une seule décision, When l’écran s’affiche, Then le nombre est annoncé au singulier', async () => {
    givenNotifications(notification(1));

    await openNotificationsAs();

    await expectCardCount(1);
    expect(announcements()).toContain(fr.loaded(1));
  });

  it('Given le vendeur connecté, When l’écran charge la liste, Then l’appel suit le contrat page 1 taille 20 avec le jeton du vendeur', async () => {
    givenNotifications(notification(1));

    await openNotificationsAs();
    await expectCardCount(1);

    expect(notificationReads().length).toBeGreaterThan(0);
    for (const request of notificationReads()) {
      const url = new URL(request.path, window.location.origin);
      expect(url.pathname).toBe('/interactions/v1/notifications');
      expect(url.searchParams.get('page')).toBe('1');
      expect(url.searchParams.get('taille')).toBe('20');
      expect(request.authorization).toMatch(/^Bearer \S+/);
    }
  });

  it('Given aucune décision, When le vendeur ouvre l’écran, Then un état vide est affiché sans liste', async () => {
    await openNotificationsAs();

    expect(await screen.findByRole('heading', { name: fr.empty })).toBeInTheDocument();
    expect(screen.queryAllByRole('listitem')).toHaveLength(0);
  });

  it('Given vingt décisions, When l’écran s’affiche, Then les vingt sont listées', async () => {
    givenNotifications(...Array.from({ length: 20 }, (_, index) => notification(index + 1)));

    await openNotificationsAs();

    await expectCardCount(20);
    expect(announcements()).toContain(fr.loaded(20));
  });
});

describe('US-028-AC2 une nouvelle notification apparaît en moins de 10 secondes', () => {
  it('Given l’écran ouvert, When une décision arrive, Then elle est listée en moins de 10 secondes sans action du vendeur', async () => {
    givenNotifications(notification(1));
    await openWithFakeTimers();
    await expectCardCount(1);

    addNotification(notification(2, { issue: 'refuse', motif: 'article_interdit' }));
    await elapse(maximumDelayMs - 1);

    await expectCardCount(2, 'fr', immediately);
    expect(cardHeadings()[0]).toBe(fr.issues.refuse);
    expect(announcements()).toContain(fr.loaded(2));
  });

  it('Given l’écran ouvert, When cinq secondes s’écoulent, Then la liste est relue', async () => {
    givenNotifications(notification(1));
    await openWithFakeTimers();
    await expectCardCount(1);
    const before = notificationReads().length;

    await elapse(pollingIntervalMs + 100);

    expect(notificationReads().length).toBeGreaterThan(before);
  });

  it('Given un écran vide, When la première décision arrive, Then l’état vide laisse place à la liste en moins de 10 secondes', async () => {
    await openWithFakeTimers();
    await screen.findByRole('heading', { name: fr.empty });

    addNotification(notification(1, { issue: 'a_verifier', motif: 'prix_a_confirmer' }));
    await elapse(maximumDelayMs - 1);

    await expectCardCount(1, 'fr', immediately);
    expect(screen.queryByRole('heading', { name: fr.empty })).not.toBeInTheDocument();
  });
});

describe('US-028-AC2 rafraîchissement suspendu quand l’onglet est masqué', () => {
  it('Given l’écran ouvert, When l’onglet est masqué, Then aucune relecture n’a lieu tant qu’il reste masqué', async () => {
    givenNotifications(notification(1));
    await openWithFakeTimers();
    await expectCardCount(1);

    hideTab(true);
    const whileHidden = notificationReads().length;
    await elapse(pollingIntervalMs * 4);

    expect(notificationReads()).toHaveLength(whileHidden);
  });

  it('Given l’onglet masqué pendant qu’une décision arrive, When le vendeur revient sur l’onglet, Then la nouvelle notification apparaît en moins de 10 secondes', async () => {
    givenNotifications(notification(1));
    await openWithFakeTimers();
    await expectCardCount(1);
    hideTab(true);
    addNotification(notification(2, { issue: 'refuse', motif: 'vendeur_non_fiable' }));
    await elapse(pollingIntervalMs * 3);
    expect(cardHeadings()).toEqual([fr.issues.publie]);

    hideTab(false);
    await elapse(maximumDelayMs - 1);

    await expectCardCount(2, 'fr', immediately);
    expect(
      within(screen.getByRole('region', { name: fr.listTitle })).getAllByRole('listitem'),
    ).toHaveLength(2);
  });
});
