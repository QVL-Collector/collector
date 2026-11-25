import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { defaultConfig, openCollector, serveConfig } from '../US-008/identityProviderHarness';
import {
  articleId,
  givenArticles,
  openAsSeller,
  storedArticle,
} from '../US-013/sellerMarketplaceHarness';
import {
  choosePhoto,
  givenPhotos,
  jpegFile,
  photoUploads,
  screenTimeout,
  storedPhoto,
  tooLarge,
} from './photoHarness';
import { fr } from './photoWording';

const id = articleId(1);
const bootstrapFailure = 'L’application n’a pas pu démarrer';

function withPhotoLimits(photos: Record<string, number>): void {
  serveConfig({ ...defaultConfig, photos });
}

async function openDraft(): Promise<void> {
  givenArticles(storedArticle(1));
  await openAsSeller(`/vendeur/articles/${id}`);
  await screen.findByRole('button', { name: fr.choose }, screenTimeout);
}

function paddedJpeg(size: number): File {
  const image = jpegFile();
  return new File([image, new Uint8Array(size - image.size)], 'lourde.jpg', { type: 'image/jpeg' });
}

describe('US-020 limites de photos lues dans config.json', () => {
  it('Given photos.maxBytes à 2 Kio, When le vendeur choisit un JPEG de 3 Kio, Then le refus pour taille s’affiche sans aucun envoi', async () => {
    withPhotoLimits({ maxBytes: 2048, maxPixels: 16000000, maxPerArticle: 1 });
    await openDraft();

    await choosePhoto(fr.choose, paddedJpeg(3072));

    expect(await screen.findByText(/^Fichier trop lourd/, {}, screenTimeout)).toBeVisible();
    expect(photoUploads().records).toHaveLength(0);
  });

  it('Given photos.maxBytes à 2 Kio, When un JPEG de 2 Kio exactement est choisi, Then il est envoyé', async () => {
    withPhotoLimits({ maxBytes: 2048, maxPixels: 16000000, maxPerArticle: 1 });
    await openDraft();

    await choosePhoto(fr.choose, paddedJpeg(2048));

    await screen.findByRole('img', { name: /Lot de cartes 1/ }, screenTimeout);
    expect(photoUploads().records).toHaveLength(1);
  });

  it('Given photos.maxPixels à 20, When le vendeur choisit un JPEG de 8 × 6 pixels, Then le refus pour dimensions s’affiche sans aucun envoi', async () => {
    withPhotoLimits({ maxBytes: 5242880, maxPixels: 20, maxPerArticle: 1 });
    await openDraft();

    await choosePhoto(fr.choose, jpegFile());

    expect(await screen.findByText(/^Image trop grande/, {}, screenTimeout)).toBeVisible();
    expect(photoUploads().records).toHaveLength(0);
  });

  it('Given photos.maxBytes à 1 Mio, When le serveur répond 413, Then le message cite la limite configurée', async () => {
    withPhotoLimits({ maxBytes: 1048576, maxPixels: 16000000, maxPerArticle: 1 });
    await openDraft();
    photoUploads().scripted.push(tooLarge());

    await choosePhoto(fr.choose, jpegFile());

    expect(
      await screen.findByText('Fichier trop lourd : 1 Mo au plus.', {}, screenTimeout),
    ).toBeVisible();
  });

  it('Given photos.maxPerArticle à 2 et une photo déposée, When l’espace s’affiche, Then un second dépôt reste proposé avec la consigne au pluriel', async () => {
    withPhotoLimits({ maxBytes: 5242880, maxPixels: 16000000, maxPerArticle: 2 });
    givenArticles(storedArticle(1));
    givenPhotos(id, storedPhoto(1));
    await openAsSeller(`/vendeur/articles/${id}`);

    expect(await screen.findByRole('button', { name: fr.choose }, screenTimeout)).toBeEnabled();
    expect(screen.getByText(/Ajoutez jusqu’à 2 photos/)).toBeVisible();
  });

  it('Given photos.maxPerArticle à 2 et deux photos, When l’espace s’affiche, Then le dépôt est remplacé par la consigne de retrait', async () => {
    withPhotoLimits({ maxBytes: 5242880, maxPixels: 16000000, maxPerArticle: 2 });
    givenArticles(storedArticle(1));
    givenPhotos(id, storedPhoto(1), storedPhoto(2));
    await openAsSeller(`/vendeur/articles/${id}`);

    expect(
      await screen.findByText(
        '2 photos au plus par article : retirez-en une pour en ajouter une autre.',
        {},
        screenTimeout,
      ),
    ).toBeVisible();
    expect(screen.queryByRole('button', { name: fr.choose })).toBeNull();
  });

  it('Given aucune limite dans config.json et une photo déposée, When l’espace s’affiche, Then le plafond par défaut d’une photo s’applique', async () => {
    givenArticles(storedArticle(1));
    givenPhotos(id, storedPhoto(1));
    await openAsSeller(`/vendeur/articles/${id}`);

    expect(
      await screen.findByText(
        'Une seule photo par article : retirez la photo actuelle pour la remplacer.',
        {},
        screenTimeout,
      ),
    ).toBeVisible();
    expect(screen.queryByRole('button', { name: fr.choose })).toBeNull();
  });

  it.each([
    ['maxPerArticle à 0', { maxBytes: 5242880, maxPixels: 16000000, maxPerArticle: 0 }],
    ['maxPerArticle à 11', { maxBytes: 5242880, maxPixels: 16000000, maxPerArticle: 11 }],
    ['maxBytes à 0', { maxBytes: 0, maxPixels: 16000000, maxPerArticle: 1 }],
    ['maxPixels négatif', { maxBytes: 5242880, maxPixels: -1, maxPerArticle: 1 }],
  ])(
    'Given photos.%s, When le front démarre, Then il affiche l’échec de démarrage',
    async (_cas, photos) => {
      withPhotoLimits(photos);

      await openCollector('/');

      expect(await screen.findByText(bootstrapFailure, {}, screenTimeout)).toBeInTheDocument();
    },
  );
});
