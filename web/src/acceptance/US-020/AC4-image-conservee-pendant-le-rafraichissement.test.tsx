import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  article,
  articleId,
  figurines,
  openAt,
  publicMarketplace,
} from '../US-023/publicMarketplaceHarness';
import { presignedUrl, screenTimeout, storedPhoto } from './photoHarness';
import { readsOf, servePublishedPhotos } from './publicPhotoHarness';

const rank = 4;
const id = articleId(rank);
const title = 'Montre Lip Dauphine';
const expiresSoonMinutes = 61 / 60;

function imagesOfTitle(): HTMLImageElement[] {
  return [...document.querySelectorAll('img')].filter((image) => image.alt === title);
}

function shownSources(): string[] {
  return imagesOfTitle()
    .filter(
      (image) =>
        getComputedStyle(image).opacity !== '0' && getComputedStyle(image).visibility !== 'hidden',
    )
    .map((image) => image.getAttribute('src') ?? '');
}

describe('US-020-AC4 URL présignée relue avant expiration, image conservée', () => {
  it('Given une photo dont l’URL expire bientôt, When la liste est relue avant expiration, Then l’image précédente reste affichée jusqu’au chargement de la nouvelle', async () => {
    publicMarketplace([article(rank, figurines, { titre: title })]);
    const server = servePublishedPhotos((_id, read) => [
      { ...storedPhoto(1, expiresSoonMinutes), url: presignedUrl(read) },
    ]);
    await openAt(`/articles/${id}`);
    const first = await screen.findByRole('img', { name: new RegExp(title) }, screenTimeout);
    fireEvent.load(first);

    await waitFor(() => {
      expect(readsOf(server, id)).toBeGreaterThanOrEqual(2);
    }, screenTimeout);

    expect(shownSources()).toContain(presignedUrl(1));
    const next = imagesOfTitle().find((image) => image.getAttribute('src') === presignedUrl(2));
    expect(next).toBeDefined();
    if (next !== undefined) {
      fireEvent.load(next);
    }
    await waitFor(() => {
      expect(shownSources()).toContain(presignedUrl(2));
    }, screenTimeout);
    expect(screen.getByRole('img', { name: new RegExp(title) })).toHaveAttribute(
      'src',
      presignedUrl(2),
    );
  });
});
