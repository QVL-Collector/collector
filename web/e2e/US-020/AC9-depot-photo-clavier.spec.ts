import { tabulerJusqua } from '../support/accessibilite';
import { expect, test } from '../support/collector';
import {
  creerUnBrouillon,
  fauxJpeg,
  imageChargee,
  photoDeLArticle,
  photoJpeg,
  svgPiege,
  textesPhotos,
} from './photos';

test.describe('US-020-AC1 AC9 dépôt de photo au clavier seul, en démonstration', () => {
  test('Given un brouillon, When le vendeur choisit, dépose puis retire une photo au clavier, Then chaque étape est opérable, annoncée et la photo a un texte alternatif', async ({
    collector,
    langue,
  }) => {
    const { page } = collector;
    const photos = textesPhotos[langue];
    const titre = `Figurine photographiée ${langue}`;
    await collector.navigateurEn(langue);
    await creerUnBrouillon(collector, titre);

    await tabulerJusqua(page, page.getByRole('button', { name: photos.choisir }));
    const selecteur = page.waitForEvent('filechooser');
    await page.keyboard.press('Enter');
    await (await selecteur).setFiles(photoJpeg);

    const photo = photoDeLArticle(page, titre);
    await expect(photo).toBeVisible();
    await expect.poll(() => imageChargee(photo)).toBe(true);
    await expect(collector.annonce(new RegExp(photos.ajoutee))).toBeVisible();

    await tabulerJusqua(page, page.getByRole('button', { name: photos.retirer }));
    await page.keyboard.press('Enter');

    await expect(photo).toHaveCount(0);
    await expect(collector.annonce(new RegExp(photos.retiree))).toBeVisible();
    await expect(page.getByRole('button', { name: photos.choisir })).toBeEnabled();
  });

  for (const [nom, fichier] of [
    ['texte nommé .jpg', fauxJpeg],
    ['SVG', svgPiege],
  ] as const) {
    test(`Given un brouillon, When le vendeur choisit un fichier ${nom}, Then le refus est expliqué et aucune photo n’apparaît`, async ({
      collector,
      langue,
    }) => {
      const { page } = collector;
      const photos = textesPhotos[langue];
      const titre = `Brouillon piégé ${langue}`;
      await collector.navigateurEn(langue);
      await creerUnBrouillon(collector, titre);

      await page
        .getByLabel(photos.choisir, { exact: true })
        .and(page.locator('input'))
        .setInputFiles(fichier);

      await expect(page.getByText(photos.nonPriseEnCharge)).toBeVisible();
      await expect(photoDeLArticle(page, titre)).toHaveCount(0);
    });
  }
});
