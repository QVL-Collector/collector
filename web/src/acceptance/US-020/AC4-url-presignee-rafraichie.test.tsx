import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { waitAtLeast } from '../US-013/sellerMarketplaceHarness';
import {
  article,
  articleId,
  figurines,
  openAt,
  publicMarketplace,
} from '../US-023/publicMarketplaceHarness';
import { presignedUrl, storedPhoto, screenTimeout } from './photoHarness';
import { fr } from './photoWording';
import { readsOf, servePublishedPhotos } from './publicPhotoHarness';

const rank = 7;
const id = articleId(rank);
const title = 'Figurine Astérix 1985';
const maximumRefreshes = 3;

async function openSheet(): Promise<void> {
  await openAt(`/articles/${id}`);
  await screen.findByRole('heading', { level: 1, name: title }, screenTimeout);
}

function shownPhoto(): HTMLImageElement | null {
  const image = screen.queryByRole('img', { name: new RegExp(title) });
  return image instanceof HTMLImageElement ? image : null;
}

async function failEveryShownPhoto(times: number): Promise<void> {
  for (let attempt = 0; attempt < times; attempt += 1) {
    const image = shownPhoto();
    if (image !== null) {
      fireEvent.error(image);
    }
    await waitAtLeast(250);
  }
}

describe('US-020-AC4 photos d’un article publié servies par une URL présignée de courte durée', () => {
  it('Given un article publié avec sa photo, When un visiteur affiche sa fiche, Then la photo est chargée depuis l’URL présignée fournie par l’API publique', async () => {
    publicMarketplace([article(rank, figurines, { titre: title })]);
    const server = servePublishedPhotos(() => [storedPhoto(1)]);

    await openSheet();

    const image = await screen.findByRole('img', { name: new RegExp(title) }, screenTimeout);
    expect(image).toHaveAttribute('src', storedPhoto(1).url);
    expect(readsOf(server, id)).toBe(1);
  });

  it('Given une URL présignée expirée, When l’image échoue à se charger, Then la liste est relue et la nouvelle URL est affichée', async () => {
    publicMarketplace([article(rank, figurines, { titre: title })]);
    const server = servePublishedPhotos((_id, read) => [
      { ...storedPhoto(1), url: presignedUrl(read) },
    ]);
    await openSheet();
    const expired = await screen.findByRole('img', { name: new RegExp(title) }, screenTimeout);

    fireEvent.error(expired);

    await waitFor(() => {
      expect(shownPhoto()?.getAttribute('src')).toBe(presignedUrl(2));
    }, screenTimeout);
    expect(readsOf(server, id)).toBe(2);
  });

  it('Given des URL qui échouent toujours, When chaque image échoue, Then les relectures restent bornées et un texte de remplacement s’affiche, sans boucle', async () => {
    publicMarketplace([article(rank, figurines, { titre: title })]);
    const server = servePublishedPhotos((_id, read) => [
      { ...storedPhoto(1), url: presignedUrl(read) },
    ]);
    await openSheet();
    await screen.findByRole('img', { name: new RegExp(title) }, screenTimeout);

    await failEveryShownPhoto(10);
    const readsAfterFailures = readsOf(server, id);
    await waitAtLeast(2000);

    expect(readsAfterFailures).toBeLessThanOrEqual(maximumRefreshes);
    expect(readsOf(server, id)).toBe(readsAfterFailures);
    expect(await screen.findByText(fr.unavailable, {}, screenTimeout)).toBeVisible();
  });

  it('Given une API qui renvoie toujours des URL déjà expirées, When la fiche reste affichée, Then elle ne relit pas la liste en boucle', async () => {
    publicMarketplace([article(rank, figurines, { titre: title })]);
    const server = servePublishedPhotos((_id, read) => [
      { ...storedPhoto(1, -1), url: presignedUrl(read) },
    ]);
    await openSheet();

    await waitAtLeast(3000);

    expect(readsOf(server, id)).toBeLessThanOrEqual(maximumRefreshes);
  });

  it('Given un article non publié, When un visiteur demande ses photos, Then la réponse 404 n’affiche aucune photo', async () => {
    publicMarketplace([article(rank, figurines, { titre: title, etat: 'brouillon' })]);
    servePublishedPhotos(() => null);

    await openAt(`/articles/${id}`);
    await screen.findByRole('heading', { level: 1 }, screenTimeout);

    expect(shownPhoto()).toBeNull();
  });
});
