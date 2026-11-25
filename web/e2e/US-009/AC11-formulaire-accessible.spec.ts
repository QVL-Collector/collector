import type { Locator, Page } from '@playwright/test';
import { violationsBloquantes } from '../support/accessibilite';
import { contenant, expect, test, type Collector } from '../support/collector';
import type { Langue } from '../support/langues';

type PriseDeFocus = { readonly invalide: string | null; readonly description: string };

type Observations = {
  readonly prisesDeFocus: { readonly element: Element; readonly etat: PriseDeFocus }[];
  rendusDeLAlerte: number;
};

type FenetreObservee = Window & { collectorE2eObservationsAc11: Observations };

async function observerLeFormulaire(page: Page): Promise<void> {
  await page.evaluate(() => {
    const observations: Observations = { prisesDeFocus: [], rendusDeLAlerte: 0 };
    (window as unknown as FenetreObservee).collectorE2eObservationsAc11 = observations;
    const texteDecrivant = (element: Element): string =>
      (element.getAttribute('aria-describedby') ?? '')
        .split(/\s+/)
        .filter((id) => id !== '')
        .map((id) => document.getElementById(id)?.textContent ?? '')
        .join(' ');
    document.addEventListener('focusin', (evenement) => {
      if (evenement.target instanceof Element) {
        observations.prisesDeFocus.push({
          element: evenement.target,
          etat: {
            invalide: evenement.target.getAttribute('aria-invalid'),
            description: texteDecrivant(evenement.target),
          },
        });
      }
    });
    const dansLAlerte = (noeud: Node): boolean =>
      (noeud instanceof Element ? noeud : noeud.parentElement)?.closest('[role="alert"]') != null;
    new MutationObserver((enregistrements) => {
      observations.rendusDeLAlerte += enregistrements.filter((enregistrement) =>
        [enregistrement.target, ...Array.from(enregistrement.addedNodes)].some(dansLAlerte),
      ).length;
    }).observe(document.body, { subtree: true, childList: true, characterData: true });
  });
}

async function premierePriseDeFocus(champ: Locator): Promise<PriseDeFocus | null> {
  return champ.evaluate(
    (element) =>
      (window as unknown as FenetreObservee).collectorE2eObservationsAc11.prisesDeFocus.find(
        (prise) => prise.element === element,
      )?.etat ?? null,
  );
}

async function rendusDeLAlerte(page: Page): Promise<number> {
  return page.evaluate(
    () => (window as unknown as FenetreObservee).collectorE2eObservationsAc11.rendusDeLAlerte,
  );
}

async function remettreAZeroLesRendusDeLAlerte(page: Page): Promise<void> {
  await page.evaluate(() => {
    (window as unknown as FenetreObservee).collectorE2eObservationsAc11.rendusDeLAlerte = 0;
  });
}

function champPrix(collector: Collector): Locator {
  return collector.page.getByRole('textbox', { name: collector.textes.champPrix });
}

async function saisirUnPrixIllisible(collector: Collector, langue: Langue): Promise<void> {
  const { page, textes } = collector;
  await collector.navigateurEn(langue);
  await collector.ouvrirEspaceProtege('/vendeur/articles/nouveau');
  await expect(collector.titrePrincipal()).toHaveText(textes.nouvelArticle);
  await page.getByRole('textbox', { name: textes.champTitre }).fill('Figurine en plomb');
  await champPrix(collector).fill('abc');
  await observerLeFormulaire(page);
}

test.describe('US-009-AC11 erreurs du formulaire de brouillon annoncées au lecteur d’écran', () => {
  test('soumission en erreur : le prix reçoit le focus déjà invalide et décrit par son erreur', async ({
    collector,
    langue,
  }) => {
    const { page, textes } = collector;
    await saisirUnPrixIllisible(collector, langue);

    await page.getByRole('button', { name: textes.creerBrouillon }).click();

    await expect(page.getByRole('alert')).toHaveText(textes.erreurFormulaire);
    await expect(champPrix(collector)).toBeFocused();
    const priseDeFocus = await premierePriseDeFocus(champPrix(collector));
    expect(priseDeFocus?.invalide).toBe('true');
    expect(priseDeFocus?.description).toContain(textes.erreurMontant);
    await expect(champPrix(collector)).toHaveAccessibleDescription(contenant(textes.erreurMontant));
  });

  test('nouvelle soumission par Entrée avec la même erreur : le résumé est de nouveau annoncé, dans une seule alerte', async ({
    collector,
    langue,
  }) => {
    const { page, textes } = collector;
    await saisirUnPrixIllisible(collector, langue);
    await page.getByRole('button', { name: textes.creerBrouillon }).click();
    await expect(page.getByRole('alert')).toHaveText(textes.erreurFormulaire);
    await remettreAZeroLesRendusDeLAlerte(page);

    await champPrix(collector).press('Enter');

    await expect.poll(() => rendusDeLAlerte(page)).toBeGreaterThan(0);
    await expect(page.getByRole('alert')).toHaveCount(1);
    await expect(page.getByRole('alert')).toHaveText(textes.erreurFormulaire);
    await expect(champPrix(collector)).toBeFocused();
  });

  test('axe-core après une seconde soumission en erreur : aucune violation serious ou critical', async ({
    collector,
    langue,
  }) => {
    const { page, textes } = collector;
    await saisirUnPrixIllisible(collector, langue);
    await page.getByRole('button', { name: textes.creerBrouillon }).click();
    await expect(page.getByRole('alert')).toHaveText(textes.erreurFormulaire);
    await champPrix(collector).press('Enter');
    await expect(page.getByRole('alert')).toHaveCount(1);

    await expect(page.locator('html')).toHaveAttribute('lang', langue);
    expect(await violationsBloquantes(page)).toEqual([]);
  });
});
