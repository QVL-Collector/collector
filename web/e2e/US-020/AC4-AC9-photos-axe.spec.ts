import { violationsBloquantes } from '../support/accessibilite';
import { articlesDemo, expect, test } from '../support/collector';
import { creerUnBrouillon, imageChargee, photoDeLArticle, photoJpeg, textesPhotos } from './photos';

test.describe('US-020-AC4 AC9 photos de la fiche publiée, en démonstration', () => {
  test('Given la fiche d’un article publié, When un visiteur l’affiche, Then chaque photo chargée a un texte alternatif et axe-core ne relève aucune violation sérieuse', async ({
    collector,
    langue,
  }) => {
    const { page } = collector;
    await collector.navigateurEn(langue);
    await collector.ouvrir(`/articles/${articlesDemo.catalogue}`);
    await expect(
      page.getByRole('region', { name: textesPhotos[langue].photosDeLaFiche }),
    ).toBeVisible();
    const titre = (await collector.titrePrincipal().textContent()) ?? '';
    expect(titre).not.toBe('');

    const photo = photoDeLArticle(page, titre).first();
    await expect(photo).toBeVisible();
    await expect.poll(() => imageChargee(photo)).toBe(true);
    const alternatifs = await page
      .getByRole('main')
      .locator('img')
      .evaluateAll((images) => images.map((image) => image.getAttribute('alt') ?? ''));
    expect(alternatifs.length).toBeGreaterThan(0);
    expect(alternatifs.every((alt) => alt.trim() !== '')).toBe(true);
    await expect(page.locator('html')).toHaveAttribute('lang', langue);
    expect(await violationsBloquantes(page)).toEqual([]);
  });
});

test.describe('US-020-AC9 axe-core sur le dépôt de photo, en démonstration', () => {
  test('Given un brouillon avec sa photo, When axe-core analyse l’écran, Then aucune violation sérieuse ou critique', async ({
    collector,
    langue,
  }) => {
    const { page } = collector;
    const photos = textesPhotos[langue];
    const titre = `Figurine axe ${langue}`;
    await collector.navigateurEn(langue);
    await creerUnBrouillon(collector, titre);
    await expect(page.locator('html')).toHaveAttribute('lang', langue);
    expect(await violationsBloquantes(page)).toEqual([]);

    await page
      .getByLabel(photos.choisir, { exact: true })
      .and(page.locator('input'))
      .setInputFiles(photoJpeg);
    await expect(photoDeLArticle(page, titre)).toBeVisible();
    await expect(page.getByRole('button', { name: photos.retirer })).toBeVisible();

    expect(await violationsBloquantes(page)).toEqual([]);
  });
});
