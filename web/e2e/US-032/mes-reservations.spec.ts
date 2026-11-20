import type { Locator } from '@playwright/test';
import { type Collector, expect, test } from '../support/collector';
import { acheteur } from '../support/fournisseurIdentite';
import { enEuros, identifiantCommande, montantsNominaux } from '../support/reservation';
import {
  carteRare,
  cheminEspaceAcheteur,
  configurerDemo,
  montreDeGousset,
  ouvrirEspaceAcheteur,
  reservationEnAttente,
  reservationExpiree,
  simulationAvec,
  textesEspaceAcheteur,
} from './espace-acheteur';

function liste(collector: Collector): Locator {
  return collector.page.getByRole('list', { name: textesEspaceAcheteur[collector.langue].liste });
}

function reservation(collector: Collector, titre: string): Locator {
  return liste(collector).getByRole('listitem').filter({ hasText: titre });
}

test.describe('US-032-AC1 l’acheteur retrouve ses réservations', () => {
  test('chaque réservation montre l’article, le montant total, l’échéance et l’état, dans l’ordre reçu', async ({
    collector,
    langue,
  }) => {
    const enAttente = reservationEnAttente(1, montreDeGousset);
    const expiree = reservationExpiree(2, carteRare);
    await ouvrirEspaceAcheteur(collector, simulationAvec(enAttente, expiree));
    const textes = textesEspaceAcheteur[langue];

    await expect(liste(collector).getByRole('listitem')).toHaveCount(2);
    await expect(liste(collector).getByRole('listitem').first()).toContainText(montreDeGousset);
    for (const [titre, commande, etat, autre] of [
      [montreDeGousset, enAttente, textes.enAttente, textes.expiree],
      [carteRare, expiree, textes.expiree, textes.enAttente],
    ] as const) {
      const element = reservation(collector, titre);
      await expect(element).toContainText(enEuros(montantsNominaux.total, langue));
      await expect(element.getByText(etat, { exact: true }).first()).toBeVisible();
      await expect(element.getByText(autre, { exact: true })).toHaveCount(0);
      const echeances = await element
        .locator('time')
        .evaluateAll((temps) =>
          temps.map((t) => new Date(t.getAttribute('datetime') ?? '').getTime()),
        );
      expect(echeances).toContain(new Date(commande.echeance).getTime());
    }
  });

  test('le lien d’une réservation ouvre son détail avec le titre de l’article', async ({
    collector,
    langue,
  }) => {
    await ouvrirEspaceAcheteur(collector, simulationAvec(reservationEnAttente(1, montreDeGousset)));

    await reservation(collector, montreDeGousset)
      .getByRole('link', { name: textesEspaceAcheteur[langue].voir(montreDeGousset) })
      .click();

    await expect(collector.page).toHaveURL(
      new RegExp(`/acheteur/commandes/${identifiantCommande(1)}$`),
    );
    await expect(collector.page.getByText(montreDeGousset).first()).toBeVisible();
  });

  test('l’entrée de navigation « Mes réservations » mène à l’espace acheteur', async ({
    collector,
    langue,
  }) => {
    const simulation = simulationAvec(reservationEnAttente(1, montreDeGousset));
    await simulation.brancher(collector);
    await collector.navigateurEn(langue);
    collector.seConnecterEn(acheteur);
    await collector.ouvrirEspaceProtege(`/acheteur/commandes/${identifiantCommande(1)}`);

    await collector.page
      .getByRole('link', { name: textesEspaceAcheteur[langue].navigation })
      .first()
      .click();

    await expect(collector.page).toHaveURL(new RegExp(`${cheminEspaceAcheteur}$`));
    await expect(collector.titrePrincipal()).toHaveText(textesEspaceAcheteur[langue].titre);
    await expect(reservation(collector, montreDeGousset)).toBeVisible();
  });

  test('sans réservation, l’état vide propose de parcourir le catalogue', async ({
    collector,
    langue,
  }) => {
    await ouvrirEspaceAcheteur(collector, simulationAvec());

    await expect(
      collector.page.getByRole('heading', { name: textesEspaceAcheteur[langue].vide }),
    ).toBeVisible();
    await expect(
      collector.page.getByRole('link', { name: textesEspaceAcheteur[langue].parcourir }),
    ).toBeVisible();
  });

  test('Transactions indisponible : erreur annoncée, puis liste affichée après « Réessayer »', async ({
    collector,
    langue,
  }) => {
    const simulation = simulationAvec(reservationEnAttente(1, montreDeGousset));
    simulation.listeIndisponible = true;
    await ouvrirEspaceAcheteur(collector, simulation);
    const textes = textesEspaceAcheteur[langue];

    await expect(collector.page.getByRole('heading', { name: textes.indisponible })).toBeVisible({
      timeout: 20_000,
    });
    simulation.listeIndisponible = false;
    await collector.page.getByRole('button', { name: textes.reessayer }).click();

    await expect(reservation(collector, montreDeGousset)).toBeVisible();
  });

  test('mode démo nominal : une liste avec au moins une réservation expirée', async ({
    collector,
    langue,
  }) => {
    await ouvrirEspaceAcheteur(collector, null);
    const textes = textesEspaceAcheteur[langue];

    await expect(liste(collector).getByRole('listitem').first()).toBeVisible();
    await expect(liste(collector).getByText(textes.expiree, { exact: true }).first()).toBeVisible();
  });

  test('mode démo vide et en erreur', async ({ collector, langue }) => {
    const textes = textesEspaceAcheteur[langue];
    await configurerDemo(collector, 'empty');
    await ouvrirEspaceAcheteur(collector, null);
    await expect(collector.page.getByRole('heading', { name: textes.vide })).toBeVisible();

    await configurerDemo(collector, 'error');
    await collector.ouvrirEspaceProtege(cheminEspaceAcheteur);
    await expect(collector.page.getByRole('heading', { name: textes.indisponible })).toBeVisible({
      timeout: 20_000,
    });
  });
});

test.describe('US-032-AC2 échéance passée pendant la consultation', () => {
  test('au rafraîchissement périodique, la réservation devient « expirée »', async ({
    collector,
    langue,
  }) => {
    await collector.page.clock.install();
    const enAttente = reservationEnAttente(
      1,
      montreDeGousset,
      new Date(Date.now() + 60_000).toISOString(),
    );
    const simulation = simulationAvec(enAttente);
    await ouvrirEspaceAcheteur(collector, simulation);
    const textes = textesEspaceAcheteur[langue];
    await expect(
      reservation(collector, montreDeGousset).getByText(textes.enAttente, { exact: true }).first(),
    ).toBeVisible();
    const listesAvant = simulation.listes.length;

    simulation.commandes = [{ ...enAttente, statut: 'expiree' }];
    await collector.page.clock.fastForward('02:00');

    await expect(
      reservation(collector, montreDeGousset).getByText(textes.expiree, { exact: true }).first(),
    ).toBeVisible();
    await expect(
      reservation(collector, montreDeGousset).getByText(textes.enAttente, { exact: true }),
    ).toHaveCount(0);
    expect(simulation.listes.length).toBeGreaterThan(listesAvant);
  });

  test('au retour sur l’onglet, la liste rafraîchie montre « expirée »', async ({
    collector,
    langue,
  }) => {
    const enAttente = reservationEnAttente(1, montreDeGousset);
    const simulation = simulationAvec(enAttente);
    await ouvrirEspaceAcheteur(collector, simulation);
    const textes = textesEspaceAcheteur[langue];
    await expect(reservation(collector, montreDeGousset)).toContainText(textes.enAttente);

    simulation.commandes = [{ ...enAttente, statut: 'expiree' }];
    await collector.page.evaluate(() => {
      window.dispatchEvent(new Event('focus'));
      window.dispatchEvent(new Event('visibilitychange'));
    });

    await expect(
      reservation(collector, montreDeGousset).getByText(textes.expiree, { exact: true }).first(),
    ).toBeVisible();
  });
});
