import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import {
  articleId,
  axeTimeout,
  blockingViolations,
  givenArticles,
  marketplace,
  openAsSeller,
  storedArticle,
} from '../US-013/sellerMarketplaceHarness';
import { article, figurines, openAt, publicMarketplace } from '../US-023/publicMarketplaceHarness';
import { fileInput, givenPhotos, photoDeletions, storedPhoto, screenTimeout } from './photoHarness';
import { fr, photoWording } from './photoWording';
import { servePublishedPhotos } from './publicPhotoHarness';

const id = articleId(1);
const title = 'Lot de cartes 1';
const languages: readonly SupportedLanguage[] = ['fr', 'en'];
const maximumTabs = 80;

async function tabUntil(target: HTMLElement): Promise<void> {
  for (let tab = 0; tab < maximumTabs && document.activeElement !== target; tab += 1) {
    await userEvent.tab();
  }
  expect(document.activeElement).toBe(target);
}

function clicksOn(element: HTMLElement): () => number {
  let clicks = 0;
  element.addEventListener('click', () => {
    clicks += 1;
  });
  return () => clicks;
}

describe('US-020-AC9 dépôt de photo opérable au clavier', () => {
  it.each(['{Enter}', ' '] as const)(
    'Given le dépôt de photo, When le vendeur atteint « Choisir une photo » à la tabulation et presse %s, Then le sélecteur de fichier s’ouvre',
    async (key) => {
      givenArticles(storedArticle(1));
      await openAsSeller(`/vendeur/articles/${id}`);
      const button = await screen.findByRole('button', { name: fr.choose }, screenTimeout);
      const pickerOpenings = clicksOn(fileInput(fr.choose));

      await tabUntil(button);
      await userEvent.keyboard(key);

      expect(pickerOpenings()).toBe(1);
    },
  );

  it('Given le champ de fichier caché, When le vendeur tabule dans l’écran, Then seul le bouton visible reçoit le focus', async () => {
    givenArticles(storedArticle(1));
    await openAsSeller(`/vendeur/articles/${id}`);
    await screen.findByRole('button', { name: fr.choose }, screenTimeout);

    expect(fileInput(fr.choose).tabIndex).toBe(-1);
  });

  it('Given une photo déposée, When le vendeur atteint « Retirer la photo » au clavier et presse Entrée, Then la photo est retirée et le focus reste dans la page', async () => {
    givenArticles(storedArticle(1));
    givenPhotos(id, storedPhoto(1));
    await openAsSeller(`/vendeur/articles/${id}`);
    const remove = await screen.findByRole('button', { name: fr.remove }, screenTimeout);

    await tabUntil(remove);
    await userEvent.keyboard('{Enter}');

    await waitFor(() => {
      expect(marketplace().photos.get(id)).toEqual([]);
    }, screenTimeout);
    expect(photoDeletions(id)).toBe(1);
    expect(document.activeElement).not.toBe(document.body);
  });
});

describe('US-020-AC9 chaque photo affichée a un texte alternatif', () => {
  it.each(languages)(
    'Given la photo d’un brouillon (%s), When l’espace vendeur l’affiche, Then son texte alternatif reprend le titre de l’article',
    async (language) => {
      givenArticles(storedArticle(1));
      givenPhotos(id, storedPhoto(1));
      await openAsSeller(`/vendeur/articles/${id}`, language);

      const image = await screen.findByRole('img', { name: new RegExp(title) }, screenTimeout);
      expect(image.getAttribute('alt')?.trim()).not.toBe('');
    },
  );

  it('Given un brouillon sans titre avec une photo, When l’espace vendeur l’affiche, Then la photo garde un texte alternatif non vide', async () => {
    givenArticles(storedArticle(1, { titre: '' }));
    givenPhotos(id, storedPhoto(1));
    await openAsSeller(`/vendeur/articles/${id}`);

    const image = await screen.findByRole('img', { name: fr.untitledAlt }, screenTimeout);
    expect(image.getAttribute('alt')).toBe(fr.untitledAlt);
  });

  it('Given un article publié avec sa photo, When un visiteur affiche sa fiche, Then la photo a un texte alternatif reprenant le titre', async () => {
    const titre = 'Montre Lip 1960';
    publicMarketplace([article(3, figurines, { titre })]);
    servePublishedPhotos(() => [storedPhoto(2)]);

    await openAt(`/articles/${articleId(3)}`);

    expect(
      await screen.findByRole('img', { name: new RegExp(titre) }, screenTimeout),
    ).toHaveAttribute('src', storedPhoto(2).url);
  });

  it.each(languages)(
    'Given l’écran d’un brouillon avec sa photo (%s), When axe-core l’analyse, Then aucune violation sérieuse ou critique',
    async (language) => {
      givenArticles(storedArticle(1));
      givenPhotos(id, storedPhoto(1));
      await openAsSeller(`/vendeur/articles/${id}`, language);
      await screen.findByRole('button', { name: photoWording[language].remove }, screenTimeout);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it(
    'Given l’écran d’un brouillon sans photo, When axe-core l’analyse, Then la zone de dépôt ne porte aucune violation sérieuse ou critique',
    async () => {
      givenArticles(storedArticle(1));
      await openAsSeller(`/vendeur/articles/${id}`);
      await screen.findByRole('button', { name: fr.choose }, screenTimeout);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );
});
