import { violationsBloquantes } from '../support/accessibilite';
import { type Collector, expect, test } from '../support/collector';
import { acheteur } from '../support/fournisseurIdentite';
import {
  articleEnVente,
  commandeEmise,
  echeanceE2e,
  enEuros,
  identifiantArticle,
  identifiantCommande,
  montantsNominaux,
  ReservationSimulee,
} from '../support/reservation';

function montantAffiche(collector: Collector, libelle: string) {
  return collector.page
    .locator('dt', { hasText: libelle })
    .locator('xpath=following-sibling::dd[1]');
}

async function ouvrirFicheEnAcheteur(
  collector: Collector,
  simulation: ReservationSimulee,
  rang: number,
): Promise<void> {
  await simulation.brancher(collector);
  await collector.navigateurEn(collector.langue);
  collector.seConnecterEn(acheteur);
  await collector.ouvrir(`/articles/${identifiantArticle(rang)}`);
  await collector.seConnecter();
  await expect(collector.page).toHaveURL(new RegExp(`/articles/${identifiantArticle(rang)}$`));
}

function simulationAvecUnArticleLibre(): ReservationSimulee {
  const simulation = new ReservationSimulee();
  simulation.articles = [articleEnVente(1)];
  simulation.prochaineCommande = commandeEmise(1, identifiantArticle(1));
  return simulation;
}

test.describe('US-030-AC1 AC2 AC6 réserver un article publié depuis sa fiche', () => {
  test('l’acheteur clique « Réserver » et arrive sur la confirmation avec les montants et l’échéance', async ({
    collector,
    langue,
  }) => {
    const { page, textes } = collector;
    const simulation = simulationAvecUnArticleLibre();
    await ouvrirFicheEnAcheteur(collector, simulation, 1);

    await page.getByRole('button', { name: textes.reserver, exact: true }).click();

    await expect(page).toHaveURL(new RegExp(`/acheteur/commandes/${identifiantCommande(1)}$`));
    await expect(collector.titrePrincipal()).toHaveText(textes.articleReserveTitre);
    await expect(page.getByText(textes.enAttenteDePaiement, { exact: true })).toBeVisible();
    await expect(montantAffiche(collector, textes.prixDeReference)).toHaveText(
      enEuros(montantsNominaux.prix, langue),
    );
    await expect(montantAffiche(collector, textes.fraisDePort)).toHaveText(
      enEuros(montantsNominaux.frais, langue),
    );
    await expect(montantAffiche(collector, textes.totalPaye)).toHaveText(
      enEuros(montantsNominaux.total, langue),
    );
    await expect(montantAffiche(collector, textes.commissionCollector)).toHaveText(
      enEuros(montantsNominaux.commission, langue),
    );
    await expect(montantAffiche(collector, textes.montantDuAuVendeur)).toHaveText(
      enEuros(montantsNominaux.vendeur, langue),
    );
    await expect(
      page.getByText(textes.aReglerAvantLe).locator('xpath=following-sibling::time[1]'),
    ).toHaveAttribute('datetime', new Date(echeanceE2e).toISOString());
  });

  test('la demande envoyée à Transactions porte le jeton de l’acheteur et seulement l’identifiant de l’article', async ({
    collector,
  }) => {
    const simulation = simulationAvecUnArticleLibre();
    await ouvrirFicheEnAcheteur(collector, simulation, 1);

    await collector.page
      .getByRole('button', { name: collector.textes.reserver, exact: true })
      .click();
    await expect(collector.titrePrincipal()).toHaveText(collector.textes.articleReserveTitre);

    expect(simulation.demandes).toHaveLength(1);
    const [demande] = simulation.demandes;
    expect(demande?.autorisation).toMatch(/^Bearer \S+/);
    expect(demande?.typeDeContenu).toMatch(/^application\/json/);
    expect(JSON.parse(demande?.corps ?? '')).toStrictEqual({ articleId: identifiantArticle(1) });
  });

  test('un visiteur non connecté ne peut pas réserver : aucune demande n’est envoyée', async ({
    collector,
  }) => {
    const simulation = simulationAvecUnArticleLibre();
    await simulation.brancher(collector);
    await collector.navigateurEn(collector.langue);
    await collector.ouvrir(`/articles/${identifiantArticle(1)}`);

    await expect(
      collector.page.getByRole('button', { name: collector.textes.connexionPourReserver }),
    ).toBeVisible();
    await expect(
      collector.page.getByRole('button', { name: collector.textes.reserver, exact: true }),
    ).toHaveCount(0);
    expect(simulation.demandes).toHaveLength(0);
  });

  test('un article devenu non réservable (422) : le refus est expliqué et l’acheteur reste sur la fiche', async ({
    collector,
  }) => {
    const simulation = simulationAvecUnArticleLibre();
    simulation.refus = { statut: 422, type: 'urn:collector:probleme:article-non-reservable' };
    await ouvrirFicheEnAcheteur(collector, simulation, 1);

    await collector.page
      .getByRole('button', { name: collector.textes.reserver, exact: true })
      .click();

    await expect(collector.page.getByText(collector.textes.articleNonReservable)).toBeVisible();
    await expect(collector.page).toHaveURL(new RegExp(`/articles/${identifiantArticle(1)}$`));
  });
});

test.describe('US-030-AC3 l’article réservé porte la mention « réservé » et n’est plus réservable', () => {
  test('catalogue : seule la carte réservée porte la mention', async ({ collector }) => {
    const simulation = new ReservationSimulee();
    simulation.articles = [articleEnVente(1, true), articleEnVente(2)];
    await simulation.brancher(collector);
    await collector.navigateurEn(collector.langue);
    await collector.ouvrir('/');

    const cartes = collector.page.getByRole('article');
    await expect(cartes).toHaveCount(2);
    await expect(
      cartes
        .filter({ hasText: 'Figurine de collection 1' })
        .getByText(collector.textes.mentionReserve),
    ).toBeVisible();
    await expect(
      cartes
        .filter({ hasText: 'Figurine de collection 2' })
        .getByText(collector.textes.mentionReserve),
    ).toHaveCount(0);
  });

  test('fiche d’un article réservé, acheteur connecté : mention, motif, aucun bouton « Réserver » actionnable', async ({
    collector,
  }) => {
    const simulation = new ReservationSimulee();
    simulation.articles = [articleEnVente(1, true)];
    await ouvrirFicheEnAcheteur(collector, simulation, 1);

    await expect(collector.page.getByText(collector.textes.articleDejaReserve)).toBeVisible();
    const reserver = collector.page.getByRole('button', {
      name: collector.textes.reserver,
      exact: true,
    });
    for (const bouton of await reserver.all()) {
      await expect(bouton).toBeDisabled();
    }
    expect(simulation.demandes).toHaveLength(0);
  });

  test('après la réservation, « Voir l’article » rouvre une fiche marquée réservée', async ({
    collector,
  }) => {
    const simulation = simulationAvecUnArticleLibre();
    await ouvrirFicheEnAcheteur(collector, simulation, 1);
    await collector.page
      .getByRole('button', { name: collector.textes.reserver, exact: true })
      .click();
    await expect(collector.titrePrincipal()).toHaveText(collector.textes.articleReserveTitre);

    await collector.page.getByRole('link', { name: collector.textes.voirLArticle }).click();

    await expect(collector.page.getByText(collector.textes.articleDejaReserve)).toBeVisible();
  });
});

test.describe('US-030 accessibilité des écrans de réservation', () => {
  test('fiche réservable, fiche réservée et confirmation : aucune violation bloquante', async ({
    collector,
  }) => {
    const simulation = new ReservationSimulee();
    simulation.articles = [articleEnVente(1), articleEnVente(2, true)];
    simulation.prochaineCommande = commandeEmise(1, identifiantArticle(1));
    await ouvrirFicheEnAcheteur(collector, simulation, 1);
    const { page, textes } = collector;
    await expect(page.getByRole('button', { name: textes.reserver, exact: true })).toBeEnabled();
    expect(await violationsBloquantes(page)).toStrictEqual([]);

    await page.getByRole('button', { name: textes.reserver, exact: true }).click();
    await expect(collector.titrePrincipal()).toHaveText(textes.articleReserveTitre);
    expect(await violationsBloquantes(page)).toStrictEqual([]);

    await page.goto(`/articles/${identifiantArticle(2)}`);
    await expect(page.getByText(textes.articleDejaReserve)).toBeVisible();
    expect(await violationsBloquantes(page)).toStrictEqual([]);
  });
});
