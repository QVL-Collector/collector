import AxeBuilder from '@axe-core/playwright';
import { expect, type Locator, type Page } from '@playwright/test';

export const tagsWcag21AA = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'];

const impactsBloquants = new Set(['serious', 'critical']);

export async function attendreRenduStable(page: Page): Promise<void> {
  await page.waitForFunction(() =>
    document
      .getAnimations()
      .filter((animation) => animation.effect?.getTiming().iterations !== Infinity)
      .every((animation) => animation.playState !== 'running'),
  );
}

export async function violationsBloquantes(page: Page): Promise<string[]> {
  await attendreRenduStable(page);
  const resultat = await new AxeBuilder({ page }).withTags(tagsWcag21AA).analyze();
  return resultat.violations
    .filter((violation) => impactsBloquants.has(violation.impact ?? ''))
    .map(
      (violation) =>
        `${violation.id} (${violation.impact ?? 'inconnu'}) : ${violation.nodes
          .map((noeud) => noeud.target.join(' '))
          .join(' | ')}`,
    );
}

export type IndicateurDeFocus = {
  readonly element: string;
  readonly indicateur: string | null;
};

export async function indicateurDeFocus(page: Page): Promise<IndicateurDeFocus> {
  return page.evaluate(() => {
    const element = document.activeElement;
    if (!(element instanceof HTMLElement) || element === document.body) {
      return { element: element?.tagName.toLowerCase() ?? 'aucun', indicateur: null };
    }
    const description = `${element.tagName.toLowerCase()} « ${(
      element.getAttribute('aria-label') ?? element.textContent
    )
      .trim()
      .slice(0, 40)} »`;
    const style = getComputedStyle(element);
    if (style.outlineStyle !== 'none' && Number.parseFloat(style.outlineWidth) > 0) {
      return {
        element: description,
        indicateur: `outline ${style.outlineWidth} ${style.outlineStyle} ${style.outlineColor}`,
      };
    }
    if (style.boxShadow !== 'none') {
      return { element: description, indicateur: `box-shadow ${style.boxShadow}` };
    }
    const champ = element.closest('.MuiInputBase-root.Mui-focused');
    const contour = champ?.querySelector('fieldset');
    if (contour instanceof HTMLElement) {
      const bordure = getComputedStyle(contour);
      if (Number.parseFloat(bordure.borderTopWidth) >= 2) {
        return {
          element: description,
          indicateur: `contour du champ ${bordure.borderTopWidth} ${bordure.borderTopColor}`,
        };
      }
    }
    return { element: description, indicateur: null };
  });
}

const tabulationsMax = 60;

export type Sens = 'avant' | 'arriere';

const touches: Readonly<Record<Sens, string>> = { avant: 'Tab', arriere: 'Shift+Tab' };

export async function tabulerJusqua(
  page: Page,
  cible: Locator,
  sens: Sens = 'avant',
): Promise<IndicateurDeFocus[]> {
  const parcourus: IndicateurDeFocus[] = [];
  for (let tabulation = 0; tabulation < tabulationsMax; tabulation += 1) {
    await page.keyboard.press(touches[sens]);
    const focus = await indicateurDeFocus(page);
    parcourus.push(focus);
    expect(
      focus.indicateur,
      `focus visible sur ${focus.element} (parcours : ${parcourus.map(({ element }) => element).join(' → ')})`,
    ).not.toBeNull();
    if (await cible.evaluate((element) => element === document.activeElement)) {
      return parcourus;
    }
  }
  throw new Error(`Cible non atteinte après ${String(tabulationsMax)} tabulations`);
}
