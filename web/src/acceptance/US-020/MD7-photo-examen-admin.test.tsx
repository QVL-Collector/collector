import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { axeTimeout, blockingViolations, waitAtLeast } from '../US-013/sellerMarketplaceHarness';
import {
  givenReviewPhotos,
  givenToReview,
  listReads,
  openAdminScreenWith,
  photoReadsOf,
  toReview,
} from '../US-022/adminReviewHarness';
import { presignedUrl, screenTimeout, storedPhoto } from './photoHarness';

const reviewed = toReview(1, { titre: 'Figurine Gaston à examiner' });
const other = toReview(2, { titre: 'Timbre sans photo' });

function cardOf(title: string): HTMLElement {
  const item = screen.getByRole('heading', { level: 3, name: title }).closest('li');
  if (item === null) {
    throw new Error(`Carte de « ${title} » introuvable`);
  }
  return item;
}

async function openWithBothCards(): Promise<void> {
  givenToReview(reviewed, other);
  await openAdminScreenWith(2);
}

describe('US-020 MD7 photo de l’article dans l’écran d’examen de l’admin', () => {
  it('Given un article à vérifier avec sa photo, When l’admin ouvre l’écran d’examen, Then la photo est lue sur la route admin avec son jeton et affichée avec le titre pour texte alternatif', async () => {
    givenReviewPhotos(reviewed.id, [storedPhoto(1)]);

    await openWithBothCards();

    const image = await within(cardOf(reviewed.titre)).findByRole(
      'img',
      { name: new RegExp(reviewed.titre) },
      screenTimeout,
    );
    expect(image).toHaveAttribute('src', storedPhoto(1).url);
    expect(photoReadsOf(reviewed.id)[0]?.authorization).toMatch(/^Bearer .+/);
  });

  it('Given un article à vérifier sans photo, When l’écran s’affiche, Then sa carte n’affiche aucune image', async () => {
    givenReviewPhotos(reviewed.id, [storedPhoto(1)]);

    await openWithBothCards();
    await within(cardOf(reviewed.titre)).findByRole('img', {}, screenTimeout);

    expect(within(cardOf(other.titre)).queryByRole('img')).toBeNull();
    expect(photoReadsOf(other.id).length).toBeGreaterThan(0);
  });

  it('Given une lecture des photos en échec, When l’écran s’affiche, Then la photo est masquée et la carte reste examinable', async () => {
    givenReviewPhotos(reviewed.id, 'echec');

    await openWithBothCards();
    await waitFor(() => {
      expect(photoReadsOf(reviewed.id).length).toBeGreaterThan(0);
    }, screenTimeout);
    await waitAtLeast(500);

    expect(within(cardOf(reviewed.titre)).queryByRole('img')).toBeNull();
    expect(
      within(cardOf(reviewed.titre)).getByRole('button', { name: /^Publier « / }),
    ).toBeEnabled();
  });

  it('Given une image qui échoue à se charger, When chaque nouvelle URL échoue aussi, Then la photo est masquée sans boucle de relecture', async () => {
    let read = 0;
    givenToReview(reviewed, other);
    givenReviewPhotos(reviewed.id, [{ ...storedPhoto(1), url: presignedUrl(1) }]);
    await openAdminScreenWith(2);
    await within(cardOf(reviewed.titre)).findByRole('img', {}, screenTimeout);

    for (let attempt = 0; attempt < 10; attempt += 1) {
      read += 1;
      givenReviewPhotos(reviewed.id, [{ ...storedPhoto(1), url: presignedUrl(read + 1) }]);
      const image = within(cardOf(reviewed.titre)).queryByRole('img');
      if (image !== null) {
        fireEvent.error(image);
      }
      await waitAtLeast(250);
    }
    const readsAfterFailures = photoReadsOf(reviewed.id).length;
    await waitAtLeast(2000);

    expect(readsAfterFailures).toBeLessThanOrEqual(3);
    expect(photoReadsOf(reviewed.id)).toHaveLength(readsAfterFailures);
    expect(within(cardOf(reviewed.titre)).queryByRole('img')).toBeNull();
  });

  it('Given des photos lues pour chaque carte, When les lectures de la liste sont comptées, Then les lectures de photos n’y figurent pas', async () => {
    givenReviewPhotos(reviewed.id, [storedPhoto(1)]);

    await openWithBothCards();
    await within(cardOf(reviewed.titre)).findByRole('img', {}, screenTimeout);

    expect(listReads().every((request) => !request.path.includes('/photos'))).toBe(true);
  });

  it(
    'Given l’écran d’examen avec une photo, When axe-core l’analyse, Then aucune violation sérieuse ou critique',
    async () => {
      givenReviewPhotos(reviewed.id, [storedPhoto(1)]);
      await openWithBothCards();
      await within(cardOf(reviewed.titre)).findByRole('img', {}, screenTimeout);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );
});
