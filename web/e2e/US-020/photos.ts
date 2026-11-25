import { expect, type Locator, type Page } from '@playwright/test';
import type { Collector } from '../support/collector';
import type { Langue } from '../support/langues';

export type TextesPhotos = {
  readonly choisir: string;
  readonly retirer: string;
  readonly ajoutee: string;
  readonly retiree: string;
  readonly nonPriseEnCharge: string;
  readonly apercu: string;
  readonly photosDeLaFiche: string;
};

export const textesPhotos: Readonly<Record<Langue, TextesPhotos>> = {
  fr: {
    choisir: 'Choisir une photo',
    retirer: 'Retirer la photo',
    ajoutee: 'Photo ajoutée.',
    retiree: 'Photo retirée.',
    nonPriseEnCharge:
      'Ce fichier n’est pas une image JPEG ou PNG valide. Les formats GIF, WebP et SVG ne sont pas acceptés.',
    apercu: 'Aperçu de la photo en cours d’envoi',
    photosDeLaFiche: 'Photos de l’article',
  },
  en: {
    choisir: 'Choose a photo',
    retirer: 'Remove photo',
    ajoutee: 'Photo added.',
    retiree: 'Photo removed.',
    nonPriseEnCharge:
      'This file is not a valid JPEG or PNG image. GIF, WebP and SVG files are not accepted.',
    apercu: 'Preview of the photo being uploaded',
    photosDeLaFiche: 'Item photos',
  },
};

const jpegBase64 =
  '/9j/2wCEAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDIBCQkJDAsMGA0NGDIhHCEyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMjIyMv/AABEIAAYACAMBIgACEQEDEQH/xAGiAAABBQEBAQEBAQAAAAAAAAAAAQIDBAUGBwgJCgsQAAIBAwMCBAMFBQQEAAABfQECAwAEEQUSITFBBhNRYQcicRQygZGhCCNCscEVUtHwJDNicoIJChYXGBkaJSYnKCkqNDU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6g4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2drh4uPk5ebn6Onq8fLz9PX29/j5+gEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoLEQACAQIEBAMEBwUEBAABAncAAQIDEQQFITEGEkFRB2FxEyIygQgUQpGhscEJIzNS8BVictEKFiQ04SXxFxgZGiYnKCkqNTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqCg4SFhoeIiYqSk5SVlpeYmZqio6Slpqeoqaqys7S1tre4ubrCw8TFxsfIycrS09TV1tfY2dri4+Tl5ufo6ery8/T19vf4+fr/2gAMAwEAAhEDEQA/AKeieB7T5eU/Kug/4Qe09U/Kruifw1v14mJx2I9o/eKyTMsV9Uj77P/Z';

export const photoJpeg = {
  name: 'objet.jpg',
  mimeType: 'image/jpeg',
  buffer: Buffer.from(jpegBase64, 'base64'),
};

export const fauxJpeg = {
  name: 'photo.jpg',
  mimeType: 'image/jpeg',
  buffer: Buffer.from('Ceci n’est pas une image'),
};

export const svgPiege = {
  name: 'logo.svg',
  mimeType: 'image/svg+xml',
  buffer: Buffer.from('<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"></svg>'),
};

export async function creerUnBrouillon(collector: Collector, titre: string): Promise<void> {
  const { page, textes } = collector;
  await collector.ouvrirEspaceProtege('/vendeur/articles/nouveau');
  await page.getByRole('textbox', { name: textes.champTitre }).fill(titre);
  await page.getByRole('textbox', { name: textes.champDescription }).fill('Socle d’origine.');
  await page.getByRole('combobox', { name: textes.champCategorie }).selectOption('figurines');
  await page.getByRole('textbox', { name: textes.champPrix }).fill(textes.prixSaisi);
  await page.getByRole('button', { name: textes.creerBrouillon }).click();
  await expect(collector.titrePrincipal()).toHaveText(titre);
}

export function photoDeLArticle(page: Page, titre: string): Locator {
  return page.getByRole('img', { name: new RegExp(titre) });
}

export async function imageChargee(image: Locator): Promise<boolean> {
  return image.evaluate(
    (element) =>
      element instanceof HTMLImageElement && element.complete && element.naturalWidth > 0,
  );
}
