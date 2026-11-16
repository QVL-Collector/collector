import { violationsBloquantes } from '../support/accessibilite';
import { type Collector, expect, test } from '../support/collector';
import { acheteur } from '../support/fournisseurIdentite';
import type { Langue } from '../support/langues';
import {
  articleEnVente,
  commandeEmise,
  echeanceE2e,
  identifiantArticle,
  identifiantCommande,
  ReservationSimulee,
} from '../support/reservation';

type TextesExpiration = {
  readonly titre: string;
  readonly statut: string;
  readonly message: string;
  readonly echeanceDepassee: string;
};

const textesExpiration: Readonly<Record<Langue, TextesExpiration>> = {
  fr: {
    titre: 'Réservation expirée',
    statut: 'Expirée',
    message: 'La réservation a expiré, l’article est de nouveau disponible.',
    echeanceDepassee: 'Échéance dépassée le',
  },
  en: {
    titre: 'Reservation expired',
    statut: 'Expired',
    message: 'The reservation has expired, the item is available again.',
    echeanceDepassee: 'Deadline passed on',
  },
};

function simulationApresEcheance(): ReservationSimulee {
  const simulation = new ReservationSimulee();
  simulation.articles = [articleEnVente(1)];
  simulation.commandes = [{ ...commandeEmise(1, identifiantArticle(1)), statut: 'expiree' }];
  simulation.prochaineCommande = {
    ...commandeEmise(2, identifiantArticle(1)),
    echeance: '2026-11-16T10:45:00.000000Z',
  };
  return simulation;
}

async function ouvrirCommandeExpiree(
  collector: Collector,
  simulation: ReservationSimulee,
): Promise<void> {
  await simulation.brancher(collector);
  await collector.navigateurEn(collector.langue);
  collector.seConnecterEn(acheteur);
  await collector.ouvrirEspaceProtege(`/acheteur/commandes/${identifiantCommande(1)}`);
  await expect(collector.titrePrincipal()).toHaveText(textesExpiration[collector.langue].titre);
}

test.describe('US-031-AC4 commande expirée et article de nouveau disponible', () => {
  test('la commande expirée montre le statut « Expirée », le message et l’échéance dépassée', async ({
    collector,
    langue,
  }) => {
    const { page, textes } = collector;
    await ouvrirCommandeExpiree(collector, simulationApresEcheance());

    await expect(page.getByText(textesExpiration[langue].statut, { exact: true })).toBeVisible();
    await expect(page.getByText(textesExpiration[langue].message)).toBeVisible();
    await expect(
      page
        .getByText(textesExpiration[langue].echeanceDepassee)
        .locator('xpath=following-sibling::time[1]'),
    ).toHaveAttribute('datetime', new Date(echeanceE2e).toISOString());
    await expect(page.getByText(textes.enAttenteDePaiement, { exact: true })).toHaveCount(0);
  });

  test('le lien vers l’article mène à sa fiche, de nouveau réservable, et un nouvel essai crée une nouvelle commande', async ({
    collector,
  }) => {
    const { page, textes } = collector;
    const simulation = simulationApresEcheance();
    await ouvrirCommandeExpiree(collector, simulation);

    await page
      .locator(`a[href$="/articles/${identifiantArticle(1)}"]`)
      .first()
      .click();

    await expect(page).toHaveURL(new RegExp(`/articles/${identifiantArticle(1)}$`));
    const reserver = page.getByRole('button', { name: textes.reserver, exact: true });
    await expect(reserver).toBeEnabled();
    await reserver.click();
    await expect(page).toHaveURL(new RegExp(`/acheteur/commandes/${identifiantCommande(2)}$`));
    await expect(collector.titrePrincipal()).toHaveText(textes.articleReserveTitre);
    expect(simulation.demandes).toHaveLength(1);
  });

  test('l’écran de la commande expirée n’a aucune violation bloquante', async ({ collector }) => {
    await ouvrirCommandeExpiree(collector, simulationApresEcheance());

    expect(await violationsBloquantes(collector.page)).toStrictEqual([]);
  });
});
