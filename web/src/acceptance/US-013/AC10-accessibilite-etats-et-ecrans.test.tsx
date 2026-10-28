import { screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { buyer, currentPath, signInAs } from '../US-008/identityProviderHarness';
import {
  articleId,
  axeTimeout,
  blockingViolations,
  click,
  concludeControl,
  expectAnnounced,
  givenArticles,
  marketplace,
  openAsSeller,
  storedArticle,
  type ArticleState,
  type ControlReason,
} from './sellerMarketplaceHarness';
import { editorHeading, field, fr, sellerWording } from './sellerScreens';
import { expectStateShown } from './sellerStatePanel';

const id = articleId(1);
const title = 'Lot de cartes 1';
const languages: readonly SupportedLanguage[] = ['fr', 'en'];

const states: readonly (readonly [ArticleState, ControlReason | null])[] = [
  ['brouillon', null],
  ['en_controle', null],
  ['publie', null],
  ['refuse', 'annonce_incomplete'],
  ['a_verifier', 'categorie_inconnue'],
];

const statesByLanguage = languages.flatMap((language) =>
  states.map(([state, reason]) => [language, state, reason] as const),
);

describe('US-013-AC10 changement d’état annoncé aux technologies d’assistance', () => {
  it('Given l’écran d’un article, When il s’affiche, Then une zone aria-live polie de rôle status est présente avant tout changement', async () => {
    givenArticles(storedArticle(1));
    await openAsSeller(`/vendeur/articles/${id}`);
    await editorHeading(title);

    const regions = screen.getAllByRole('status');
    expect(regions.some((region) => region.getAttribute('aria-live') === 'polite')).toBe(true);
  });

  it.each(['publie', 'refuse', 'a_verifier'] as const)(
    'Given un article en contrôle, When il devient %s, Then le nouvel état est annoncé dans la zone status',
    async (outcome) => {
      givenArticles(storedArticle(1));
      await openAsSeller(`/vendeur/articles/${id}`);
      await editorHeading(title);
      await click(screen.getByRole('button', { name: fr.submit }));
      await expectAnnounced(fr.states.en_controle);

      concludeControl(
        id,
        outcome,
        outcome === 'publie'
          ? null
          : outcome === 'refuse'
            ? 'annonce_incomplete'
            : 'categorie_inconnue',
      );

      await expectAnnounced(fr.states[outcome], 6000);
    },
  );
});

describe('US-013-AC10 axe-core sur les trois écrans vendeur, en français et en anglais', () => {
  it.each(languages)(
    'Given « Mes articles » avec un article par état (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      givenArticles(
        ...states.map(([etat, motif], rank) => storedArticle(rank + 1, { etat, motif })),
      );
      await openAsSeller('/vendeur', language);
      await screen.findByRole('list', { name: sellerWording[language].myItems });

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(languages)(
    'Given « Mes articles » vide (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      await openAsSeller('/vendeur', language);
      await screen.findByRole('link', { name: sellerWording[language].newItem });

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(languages)(
    'Given la création d’un article (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      await openAsSeller('/vendeur/articles/nouveau', language);
      await editorHeading(sellerWording[language].newItemTitle);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(statesByLanguage)(
    'Given l’écran d’un article (%s) à l’état %s, When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language, etat, motif) => {
      givenArticles(storedArticle(1, { etat, motif }));
      await openAsSeller(`/vendeur/articles/${id}`, language);
      await editorHeading(title);
      await expectStateShown(sellerWording[language].states[etat], language);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(languages)(
    'Given le conflit 409 expliqué (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      const wording = sellerWording[language];
      givenArticles(storedArticle(1));
      await openAsSeller(`/vendeur/articles/${id}`, language);
      await editorHeading(title);
      marketplace().articles.set(id, storedArticle(1, { revision: 2, titre: 'Autre version' }));
      await click(screen.getByRole('button', { name: wording.save }));
      await screen.findByRole('button', { name: wording.reload });

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );
});

describe('US-013-AC10 parcours au clavier', () => {
  it('Given « Mes articles », When le vendeur tabule jusqu’à « Nouvel article » et valide, Then il arrive sur le formulaire de création', async () => {
    await openAsSeller('/vendeur');
    const newItem = await screen.findByRole('link', { name: fr.newItem });

    for (let step = 0; step < 30 && document.activeElement !== newItem; step += 1) {
      await userEvent.tab();
    }
    expect(newItem).toHaveFocus();
    await userEvent.keyboard('{Enter}');

    await editorHeading(fr.newItemTitle);
  });

  it('Given le formulaire de création, When le vendeur saisit au clavier et valide « Créer le brouillon », Then le brouillon est créé', async () => {
    await openAsSeller('/vendeur/articles/nouveau');
    await editorHeading(fr.newItemTitle);

    await userEvent.click(field(fr.title));
    await userEvent.keyboard('Timbre rare');
    const create = screen.getByRole('button', { name: fr.create });
    for (let step = 0; step < 30 && document.activeElement !== create; step += 1) {
      await userEvent.tab();
    }
    expect(create).toHaveFocus();
    await userEvent.keyboard('{Enter}');

    await editorHeading('Timbre rare');
  });

  it('Given un brouillon, When le vendeur atteint « Soumettre au contrôle » par tabulation et l’active à la barre d’espace, Then l’article passe en contrôle', async () => {
    givenArticles(storedArticle(1));
    await openAsSeller(`/vendeur/articles/${id}`);
    await editorHeading(title);
    const submit = screen.getByRole('button', { name: fr.submit });

    for (let step = 0; step < 40 && document.activeElement !== submit; step += 1) {
      await userEvent.tab();
    }
    expect(submit).toHaveFocus();
    await userEvent.keyboard(' ');

    await expectStateShown(fr.states.en_controle);
  });
});

describe('US-013 front, accès refusé à un non-vendeur', () => {
  it.each(['/vendeur', '/vendeur/articles/nouveau', `/vendeur/articles/${id}`])(
    'Given un acheteur connecté, When il ouvre %s, Then l’accès est refusé sans appel à l’API vendeur',
    async (path) => {
      givenArticles(storedArticle(1));
      await signInAs(buyer, path);

      expect(
        await screen.findByRole('heading', { level: 1, name: fr.accessDenied }),
      ).toBeInTheDocument();
      expect(currentPath()).toBe(path);
      expect(
        marketplace().requests.filter((request) => request.path.includes('/vendeur/')),
      ).toEqual([]);
    },
  );
});
