import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import {
  articleId,
  axeTimeout,
  blockingViolations,
  click,
  givenArticles,
  invalidInput,
  marketplace,
  openAsSeller,
  storedArticle,
  type,
  writesTo,
} from '../US-013/sellerMarketplaceHarness';
import { categoryField, editorHeading, field, fr, sellerWording } from '../US-013/sellerScreens';

const id = articleId(1);
const languages: readonly SupportedLanguage[] = ['fr', 'en'];

const errorWording: Readonly<
  Record<
    SupportedLanguage,
    { negative: string; tooLong: string; unknownCategory: string; summary: string }
  >
> = {
  fr: {
    negative: 'Le montant ne peut pas être négatif.',
    tooLong: 'Texte trop long.',
    unknownCategory: 'Catégorie inconnue.',
    summary: 'Le formulaire contient 3 erreurs : corrigez les champs signalés.',
  },
  en: {
    negative: 'The amount cannot be negative.',
    tooLong: 'Text too long.',
    unknownCategory: 'Unknown category.',
    summary: 'The form contains 3 errors: fix the highlighted fields.',
  },
};

const threeFieldErrors = invalidInput([
  { champ: 'titre', code: 'trop_long' },
  { champ: 'categorieId', code: 'categorie_inconnue' },
  { champ: 'prix.montantCentimes', code: 'negatif' },
]);

async function saveDraftRejectedWith(language: SupportedLanguage): Promise<void> {
  const wording = sellerWording[language];
  givenArticles(storedArticle(1));
  await openAsSeller(`/vendeur/articles/${id}`, language);
  await editorHeading('Lot de cartes 1');
  marketplace().scriptedWrites.push(threeFieldErrors);
  await type(field(wording.title), 'Lot de cartes modifié');
  await click(screen.getByRole('button', { name: wording.save }));
  await screen.findByRole('alert');
}

describe('US-009-AC11 chaque champ du formulaire a un libellé', () => {
  it.each(languages)(
    'Given le formulaire de création (%s), When il s’affiche, Then titre, description, catégorie, prix et frais de port sont nommés',
    async (language) => {
      const wording = sellerWording[language];
      await openAsSeller('/vendeur/articles/nouveau', language);
      await editorHeading(wording.newItemTitle);

      for (const name of [
        wording.title,
        wording.description,
        wording.price,
        wording.shippingCost,
      ]) {
        expect(field(name)).toBeInTheDocument();
      }
      expect(categoryField(wording.category)).toBeInTheDocument();
    },
  );
});

describe('US-009-AC11 et AC4 erreurs 400 de l’API rattachées aux champs et annoncées', () => {
  it('Given une saisie refusée en 400 par l’API, When la réponse arrive, Then chaque erreur est affichée sous son champ et liée à lui', async () => {
    await saveDraftRejectedWith('fr');

    expect(field(fr.title)).toHaveAttribute('aria-invalid', 'true');
    expect(field(fr.title)).toHaveAccessibleDescription(
      expect.stringContaining('Texte trop long.') as string,
    );
    expect(field(fr.price)).toHaveAttribute('aria-invalid', 'true');
    expect(field(fr.price)).toHaveAccessibleDescription(
      expect.stringContaining('Le montant ne peut pas être négatif.') as string,
    );
    expect(categoryField()).toHaveAttribute('aria-invalid', 'true');
    expect(categoryField()).toHaveAccessibleDescription(
      expect.stringContaining('Catégorie inconnue.') as string,
    );
    expect(field(fr.description)).toHaveAttribute('aria-invalid', 'false');
  });

  it.each(languages)(
    'Given une saisie refusée en 400 (%s), When la réponse arrive, Then un résumé est annoncé en alerte et le focus va au premier champ en erreur',
    async (language) => {
      await saveDraftRejectedWith(language);

      expect(screen.getByRole('alert')).toHaveTextContent(errorWording[language].summary);
      await waitFor(() => {
        expect(field(sellerWording[language].title)).toHaveFocus();
      });
    },
  );

  it('Given une seule erreur sur le prix, When elle est annoncée, Then le résumé est au singulier et le focus va au prix', async () => {
    givenArticles(storedArticle(1));
    await openAsSeller(`/vendeur/articles/${id}`);
    await editorHeading('Lot de cartes 1');
    marketplace().scriptedWrites.push(
      invalidInput([{ champ: 'prix.montantCentimes', code: 'au_dela_du_plafond' }]),
    );

    await type(field(fr.price), '2000000');
    await click(screen.getByRole('button', { name: fr.save }));

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Le formulaire contient 1 erreur : corrigez le champ signalé.',
    );
    expect(field(fr.price)).toHaveAccessibleDescription(
      expect.stringContaining('Montant au-delà du plafond autorisé.') as string,
    );
    await waitFor(() => {
      expect(field(fr.price)).toHaveFocus();
    });
  });

  it('Given une création refusée en 400, When la réponse arrive, Then l’erreur est rattachée au champ et aucun article n’est créé', async () => {
    await openAsSeller('/vendeur/articles/nouveau');
    await editorHeading(fr.newItemTitle);
    marketplace().scriptedWrites.push(
      invalidInput([{ champ: 'titre', code: 'caractere_interdit' }]),
    );

    await type(field(fr.title), 'Titre');
    await click(screen.getByRole('button', { name: fr.create }));

    expect(await screen.findByRole('alert')).toBeInTheDocument();
    expect(field(fr.title)).toHaveAccessibleDescription(
      expect.stringContaining('Ce texte contient un caractère non autorisé.') as string,
    );
    expect(marketplace().articles.size).toBe(0);
    expect(writesTo('POST', '/vendeur/articles')).toHaveLength(1);
  });

  it('Given des erreurs affichées, When le vendeur corrige et enregistre avec succès, Then les erreurs disparaissent', async () => {
    await saveDraftRejectedWith('fr');

    await type(field(fr.title), 'Lot corrigé');
    await click(screen.getByRole('button', { name: fr.save }));

    await waitFor(() => {
      expect(field(fr.title)).toHaveAttribute('aria-invalid', 'false');
    });
    expect(screen.queryByText('Texte trop long.')).not.toBeInTheDocument();
  });
});

describe('US-009-AC11 axe-core sur le formulaire', () => {
  it.each(languages)(
    'Given le formulaire de création vide (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      await openAsSeller('/vendeur/articles/nouveau', language);
      await editorHeading(sellerWording[language].newItemTitle);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(languages)(
    'Given le formulaire affichant des erreurs de l’API (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      await saveDraftRejectedWith(language);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );
});
