import { violationsBloquantes } from '../support/accessibilite';
import { expect, test } from '../support/collector';
import { ecrans } from '../support/ecrans';

test.describe('US-025-AC4 axe-core WCAG 2.1 A et AA : aucune violation serious ou critical', () => {
  for (const ecran of ecrans) {
    test(ecran.nom, async ({ collector, langue }) => {
      await collector.navigateurEn(langue);
      await ecran.ouvrir(collector);
      await expect(collector.page.locator('html')).toHaveAttribute('lang', langue);
      expect(await violationsBloquantes(collector.page)).toEqual([]);
    });
  }
});
