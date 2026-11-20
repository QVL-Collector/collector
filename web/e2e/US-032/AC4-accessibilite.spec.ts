import { violationsBloquantes } from '../support/accessibilite';
import { expect, test } from '../support/collector';
import {
  configurerDemo,
  montreDeGousset,
  carteRare,
  ouvrirEspaceAcheteur,
  reservationEnAttente,
  reservationExpiree,
  simulationAvec,
  textesEspaceAcheteur,
} from './espace-acheteur';

test.describe('US-032-AC4 la liste des réservations n’a aucune violation axe-core serious ou critical', () => {
  test('liste avec une réservation en attente et une expirée', async ({ collector, langue }) => {
    await ouvrirEspaceAcheteur(
      collector,
      simulationAvec(reservationEnAttente(1, montreDeGousset), reservationExpiree(2, carteRare)),
    );
    await expect(
      collector.page
        .getByRole('list', { name: textesEspaceAcheteur[langue].liste })
        .getByRole('listitem'),
    ).toHaveCount(2);

    expect(await violationsBloquantes(collector.page)).toStrictEqual([]);
  });

  test('liste du mode démo', async ({ collector, langue }) => {
    await ouvrirEspaceAcheteur(collector, null);
    await expect(
      collector.page
        .getByRole('list', { name: textesEspaceAcheteur[langue].liste })
        .getByRole('listitem')
        .first(),
    ).toBeVisible();

    expect(await violationsBloquantes(collector.page)).toStrictEqual([]);
  });

  test('état vide', async ({ collector, langue }) => {
    await ouvrirEspaceAcheteur(collector, simulationAvec());
    await expect(
      collector.page.getByRole('heading', { name: textesEspaceAcheteur[langue].vide }),
    ).toBeVisible();

    expect(await violationsBloquantes(collector.page)).toStrictEqual([]);
  });

  test('état d’erreur', async ({ collector, langue }) => {
    await configurerDemo(collector, 'error');
    await ouvrirEspaceAcheteur(collector, null);
    await expect(
      collector.page.getByRole('heading', { name: textesEspaceAcheteur[langue].indisponible }),
    ).toBeVisible({
      timeout: 20_000,
    });

    expect(await violationsBloquantes(collector.page)).toStrictEqual([]);
  });
});
