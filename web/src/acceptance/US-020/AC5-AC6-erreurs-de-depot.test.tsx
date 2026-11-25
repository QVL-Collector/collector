import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { ScriptedAnswer } from '../US-013/sellerMarketplaceHarness';
import {
  articleId,
  givenArticles,
  marketplace,
  openAsSeller,
  storedArticle,
} from '../US-013/sellerMarketplaceHarness';
import {
  choosePhoto,
  gifFile,
  jpegFile,
  limitReached,
  notModifiable,
  photoUploads,
  svgFile,
  textNamedJpeg,
  tooLarge,
  tooManyPixels,
  unsupported,
  webpFile,
  screenTimeout,
} from './photoHarness';
import { fr } from './photoWording';

const id = articleId(1);
const title = 'Lot de cartes 1';
const maximumBytes = 5 * 1024 * 1024;

async function openDraft(): Promise<void> {
  givenArticles(storedArticle(1));
  await openAsSeller(`/vendeur/articles/${id}`);
  await screen.findByRole('button', { name: fr.choose }, screenTimeout);
}

function expectNoPhotoStoredOrShown(): void {
  expect(marketplace().photos.get(id) ?? []).toEqual([]);
  expect(screen.queryByRole('img', { name: new RegExp(title) })).toBeNull();
}

function oversizedJpeg(): File {
  return new File([jpegFile(), new Uint8Array(maximumBytes)], 'lourde.jpg', { type: 'image/jpeg' });
}

describe('US-020-AC5 fichier qui n’est pas une image JPEG ou PNG refusé', () => {
  it.each([
    ['GIF', gifFile],
    ['WebP', webpFile],
    ['SVG', svgFile],
    ['texte portant l’extension .jpg', textNamedJpeg],
  ] as const)(
    'Given un brouillon, When le vendeur choisit un fichier %s, Then le refus est expliqué et aucune photo n’est enregistrée',
    async (_kind, file) => {
      await openDraft();

      await choosePhoto(fr.choose, file());

      expect(await screen.findByText(fr.unsupported, {}, screenTimeout)).toBeVisible();
      expectNoPhotoStoredOrShown();
    },
  );
});

describe('US-020-AC5 AC6 refus du serveur expliqué au vendeur', () => {
  it.each([
    ['415 contenu non pris en charge', unsupported, fr.unsupported],
    ['413 au-delà de la taille maximale', tooLarge, fr.tooLarge],
    ['422 dimensions excessives', tooManyPixels, fr.tooManyPixels],
    ['409 plafond de photos atteint', limitReached, fr.limitReached],
    ['409 article non modifiable', notModifiable, fr.locked],
  ] as const satisfies readonly (readonly [string, () => ScriptedAnswer, string])[])(
    'Given un JPEG d’apparence valide, When le serveur répond %s, Then le message dédié s’affiche, aucune photo n’apparaît et un nouvel essai reste possible',
    async (_answer, answer, message) => {
      await openDraft();
      photoUploads().scripted.push(answer());

      await choosePhoto(fr.choose, jpegFile());

      expect(await screen.findByText(message, {}, screenTimeout)).toBeVisible();
      expect(photoUploads().records).toHaveLength(1);
      expectNoPhotoStoredOrShown();
      expect(screen.getByRole('button', { name: fr.choose })).toBeEnabled();
    },
  );

  it('Given un brouillon, When le vendeur choisit un fichier au-delà de 5 Mo, Then le refus pour taille est expliqué sans photo enregistrée', async () => {
    await openDraft();

    await choosePhoto(fr.choose, oversizedJpeg());

    expect(await screen.findByText(fr.tooLarge, {}, screenTimeout)).toBeVisible();
    expectNoPhotoStoredOrShown();
  });

  it('Given un refus 415, When le vendeur choisit ensuite un JPEG valide, Then le message d’erreur disparaît et la photo s’affiche', async () => {
    await openDraft();
    photoUploads().scripted.push(unsupported());
    await choosePhoto(fr.choose, jpegFile());
    await screen.findByText(fr.unsupported, {}, screenTimeout);

    await choosePhoto(fr.choose, jpegFile());

    await screen.findByRole('img', { name: new RegExp(title) }, screenTimeout);
    await waitFor(() => {
      expect(screen.queryByText(fr.unsupported)).toBeNull();
    }, screenTimeout);
  });
});
