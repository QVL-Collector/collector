import type { Page } from '@playwright/test';
import { violationsBloquantes } from '../support/accessibilite';
import { type Collector, expect, test } from '../support/collector';
import { acheteur } from '../support/fournisseurIdentite';
import { delaiMaximalMs, InteractionsSimulees, notificationEmise } from '../support/interactions';

const cheminNotifications = '/vendeur/notifications';

function listeDesDecisions(collector: Collector) {
  return collector.page
    .getByRole('region', { name: collector.textes.decisionsSurVosArticles })
    .getByRole('list');
}

async function masquerOnglet(page: Page, masque: boolean): Promise<void> {
  await page.evaluate((etat) => {
    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      get: () => (etat ? 'hidden' : 'visible'),
    });
    Object.defineProperty(document, 'hidden', { configurable: true, get: () => etat });
    document.dispatchEvent(new Event('visibilitychange'));
  }, masque);
}

async function ouvrirAvec(collector: Collector, interactions: InteractionsSimulees): Promise<void> {
  await interactions.brancher(collector);
  await collector.navigateurEn(collector.langue);
  await collector.ouvrirEspaceProtege(cheminNotifications);
}

test.describe('US-028-AC2 notifications listées dans l’espace vendeur', () => {
  test('connexion vendeur, lien « Notifications », liste des décisions de démonstration', async ({
    collector,
    langue,
  }) => {
    const { page, textes } = collector;
    await collector.ouvrir('/');
    await collector.choisirLangue(langue);
    await collector.seConnecter();

    await page.getByRole('banner').getByRole('link', { name: textes.notifications }).click();

    await expect(page).toHaveURL(new RegExp(`${cheminNotifications}$`));
    await expect(collector.titrePrincipal()).toHaveText(textes.notifications);
    const liste = listeDesDecisions(collector);
    await expect(liste.getByRole('listitem')).toHaveCount(3);
    await expect(liste.getByRole('heading', { level: 3 })).toHaveText([
      textes.notificationPubliee,
      textes.notificationAVerifier,
      textes.notificationRefusee,
    ]);
    await expect(collector.annonce(new RegExp(textes.nombreDeNotifications(3)))).toBeAttached();
  });

  test('une décision émise pendant la consultation apparaît en moins de 10 secondes', async ({
    collector,
  }) => {
    const interactions = new InteractionsSimulees();
    interactions.notifications = [notificationEmise(1)];
    await ouvrirAvec(collector, interactions);
    const liste = listeDesDecisions(collector);
    await expect(liste.getByRole('listitem')).toHaveCount(1);

    interactions.recevoir(notificationEmise(2, 'refuse', 'article_interdit'));
    const emiseA = Date.now();

    await expect(liste.getByRole('listitem')).toHaveCount(2, { timeout: delaiMaximalMs });
    expect(Date.now() - emiseA).toBeLessThan(delaiMaximalMs);
    await expect(liste.getByRole('heading', { level: 3 }).first()).toHaveText(
      collector.textes.notificationRefusee,
    );
  });

  test('l’appel suit le contrat page 1 taille 20 avec le jeton du vendeur', async ({
    collector,
  }) => {
    const interactions = new InteractionsSimulees();
    interactions.notifications = [notificationEmise(1)];
    await ouvrirAvec(collector, interactions);
    await expect(listeDesDecisions(collector).getByRole('listitem')).toHaveCount(1);

    expect(interactions.lectures.length).toBeGreaterThan(0);
    for (const lecture of interactions.lectures) {
      expect(lecture.url.pathname).toBe('/interactions/v1/notifications');
      expect(lecture.url.searchParams.get('page')).toBe('1');
      expect(lecture.url.searchParams.get('taille')).toBe('20');
      expect(lecture.autorisation).toMatch(/^Bearer \S+/);
    }
  });

  test('onglet masqué : aucune relecture ; retour sur l’onglet : la décision arrivée entre-temps apparaît en moins de 10 secondes', async ({
    collector,
  }) => {
    const { page } = collector;
    const interactions = new InteractionsSimulees();
    interactions.notifications = [notificationEmise(1)];
    await ouvrirAvec(collector, interactions);
    const liste = listeDesDecisions(collector);
    await expect(liste.getByRole('listitem')).toHaveCount(1);

    await masquerOnglet(page, true);
    const lecturesAuMasquage = interactions.lectures.length;
    interactions.recevoir(notificationEmise(2, 'a_verifier', 'categorie_inconnue'));
    await page.waitForTimeout(12_000);
    expect(interactions.lectures).toHaveLength(lecturesAuMasquage);
    await expect(liste.getByRole('listitem')).toHaveCount(1);

    await masquerOnglet(page, false);

    await expect(liste.getByRole('listitem')).toHaveCount(2, { timeout: delaiMaximalMs });
  });
});

test.describe('US-028-AC4 texte construit sans contenu saisi par un utilisateur', () => {
  test('le motif est traduit, le code brut et le titre de l’article n’apparaissent pas', async ({
    collector,
  }) => {
    const interactions = new InteractionsSimulees();
    interactions.notifications = [
      notificationEmise(2, 'a_verifier', 'categorie_inconnue'),
      notificationEmise(1, 'refuse', 'annonce_incomplete'),
    ];
    await ouvrirAvec(collector, interactions);
    const liste = listeDesDecisions(collector);
    await expect(liste.getByRole('listitem')).toHaveCount(2);

    const aVerifier = liste.getByRole('listitem').nth(0);
    const refusee = liste.getByRole('listitem').nth(1);
    await expect(aVerifier).toContainText(collector.textes.notificationAVerifier);
    await expect(aVerifier).toContainText(collector.textes.motifCategorieInconnue);
    await expect(refusee).toContainText(collector.textes.notificationRefusee);
    await expect(refusee).toContainText(collector.textes.motifAnnonceIncomplete);
    const texte = (await liste.textContent()) ?? '';
    expect(texte).not.toContain('categorie_inconnue');
    expect(texte).not.toContain('annonce_incomplete');
  });

  test('les décisions de démonstration ne reprennent aucun titre d’article du vendeur', async ({
    collector,
  }) => {
    const { page } = collector;
    await collector.navigateurEn(collector.langue);
    await collector.ouvrirEspaceProtege('/vendeur');
    await expect(page.getByText(collector.textes.motifIncomplet)).toBeVisible();
    const titresDuVendeur = (await page.getByRole('main').getByRole('heading').allTextContents())
      .map((titre) => titre.trim())
      .filter((titre) => titre !== '' && titre !== collector.textes.espaceVendeur);

    await page
      .getByRole('banner')
      .getByRole('link', { name: collector.textes.notifications })
      .click();
    const liste = listeDesDecisions(collector);
    await expect(liste.getByRole('listitem')).toHaveCount(3);

    const texte = (await page.getByRole('main').textContent()) ?? '';
    expect(titresDuVendeur.length).toBeGreaterThan(0);
    for (const titre of titresDuVendeur) {
      expect(texte).not.toContain(titre);
    }
  });
});

test.describe('US-028-AC6 côté front : 401 et 403', () => {
  test('401 : état « session expirée » avec le bouton pour se reconnecter', async ({
    collector,
  }) => {
    const interactions = new InteractionsSimulees();
    interactions.notifications = [notificationEmise(1)];
    interactions.statut = 401;
    await ouvrirAvec(collector, interactions);

    await expect(
      collector.page.getByRole('heading', { name: collector.textes.sessionExpiree }),
    ).toBeVisible();
    await expect(
      collector.page.getByRole('button', { name: collector.textes.seReconnecter }),
    ).toBeVisible();
    await expect(collector.page.getByRole('listitem')).toHaveCount(0);
  });

  test('403 : état « accès réservé » sans notification', async ({ collector }) => {
    const interactions = new InteractionsSimulees();
    interactions.notifications = [notificationEmise(1)];
    interactions.statut = 403;
    await ouvrirAvec(collector, interactions);

    await expect(
      collector.page.getByRole('heading', { name: collector.textes.accesReserve }),
    ).toBeVisible();
    await expect(collector.page.getByText(collector.textes.notificationPubliee)).toHaveCount(0);
  });

  test('compte sans rôle vendeur : « accès réservé » et axe-core sans violation bloquante', async ({
    collector,
  }) => {
    collector.seConnecterEn(acheteur);
    await collector.navigateurEn(collector.langue);
    await collector.ouvrirEspaceProtege(cheminNotifications);

    await expect(collector.titrePrincipal()).toHaveText(collector.textes.accesReserve);
    expect(await violationsBloquantes(collector.page)).toEqual([]);
  });
});
