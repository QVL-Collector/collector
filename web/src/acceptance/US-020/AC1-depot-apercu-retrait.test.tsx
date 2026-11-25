import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  articleId,
  click,
  expectAnnounced,
  givenArticles,
  marketplace,
  openAsSeller,
  storedArticle,
} from '../US-013/sellerMarketplaceHarness';
import {
  choosePhoto,
  givenPhotos,
  holdUploads,
  jpegFile,
  photoDeletions,
  photoUploads,
  pngFile,
  storedPhoto,
  announcementTimeout,
  screenTimeout,
} from './photoHarness';
import { fr } from './photoWording';

const id = articleId(1);
const title = 'Lot de cartes 1';

async function openDraft(): Promise<void> {
  givenArticles(storedArticle(1));
  await openAsSeller(`/vendeur/articles/${id}`);
  await screen.findByRole('button', { name: fr.choose }, screenTimeout);
}

async function displayedPhoto(): Promise<HTMLImageElement> {
  const image = await screen.findByRole('img', { name: new RegExp(title) }, screenTimeout);
  if (!(image instanceof HTMLImageElement)) {
    throw new Error('Photo affichée sans élément image');
  }
  return image;
}

describe('US-020-AC1 dépôt d’une photo JPEG ou PNG sur un brouillon', () => {
  it.each([
    ['JPEG', jpegFile, 'image/jpeg'],
    ['PNG', pngFile, 'image/png'],
  ] as const)(
    'Given un brouillon, When le vendeur dépose une photo %s, Then le fichier brut est envoyé avec son jeton et la photo s’affiche dans son espace',
    async (_format, file, mediaType) => {
      await openDraft();
      const chosen = file();

      await choosePhoto(fr.choose, chosen);

      const image = await displayedPhoto();
      expect(image.src).toBe(marketplace().photos.get(id)?.[0]?.url);
      expect(photoUploads().records).toEqual([
        {
          path: `/marche/v1/vendeur/articles/${id}/photos`,
          authorization: expect.stringMatching(/^Bearer .+/) as unknown,
          contentType: mediaType,
          size: chosen.size,
        },
      ]);
      await expectAnnounced(fr.added, announcementTimeout);
    },
  );

  it('Given un envoi en cours, When le fichier part, Then un aperçu du fichier choisi s’affiche avec son texte alternatif, puis la photo enregistrée le remplace', async () => {
    await openDraft();
    const release = holdUploads();

    await choosePhoto(fr.choose, jpegFile());

    const preview = await screen.findByRole('img', { name: fr.previewAlt }, screenTimeout);
    expect(preview.getAttribute('src')).toBe('blob:apercu-photo');
    release();
    await displayedPhoto();
    await waitFor(() => {
      expect(screen.queryByRole('img', { name: fr.previewAlt })).toBeNull();
    }, screenTimeout);
  });

  it('Given un brouillon avec sa photo, When le vendeur la retire, Then la photo disparaît de son espace, le retrait est annoncé et un nouveau dépôt est proposé', async () => {
    givenArticles(storedArticle(1));
    givenPhotos(id, storedPhoto(1));
    await openAsSeller(`/vendeur/articles/${id}`);
    await displayedPhoto();

    await click(await screen.findByRole('button', { name: fr.remove }, screenTimeout));

    await waitFor(() => {
      expect(screen.queryByRole('img', { name: new RegExp(title) })).toBeNull();
    }, screenTimeout);
    expect(photoDeletions(id)).toBe(1);
    expect(marketplace().photos.get(id)).toEqual([]);
    await expectAnnounced(fr.removed, announcementTimeout);
    expect(await screen.findByRole('button', { name: fr.choose }, screenTimeout)).toBeEnabled();
  });

  it('Given un brouillon déjà pourvu de sa photo, When l’espace s’affiche, Then la photo existante est affichée depuis son URL présignée', async () => {
    givenArticles(storedArticle(1));
    givenPhotos(id, storedPhoto(3));
    await openAsSeller(`/vendeur/articles/${id}`);

    expect((await displayedPhoto()).src).toBe(storedPhoto(3).url);
  });
});
