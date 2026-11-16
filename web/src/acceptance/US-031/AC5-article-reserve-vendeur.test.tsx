import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import {
  articleId,
  click,
  givenArticles,
  marketplace,
  openAsSeller,
  storedArticle,
  type,
  writesTo,
  type ArticleState,
  type ScriptedAnswer,
} from '../US-013/sellerMarketplaceHarness';
import { editorHeading, field, sellerWording } from '../US-013/sellerScreens';
import { liveRegionsText } from '../US-013/sellerStatePanel';

const id = articleId(1);
const title = 'Lot de cartes 1';
const languages: readonly SupportedLanguage[] = ['fr', 'en'];

const reservedMessage: Readonly<Record<SupportedLanguage, string>> = {
  fr: 'Cet article est réservé par un acheteur ou déjà vendu : il ne peut être ni modifié ni soumis pendant la réservation, ni après la vente.',
  en: 'This item is reserved by a buyer or already sold: it cannot be changed or submitted during the reservation, nor after the sale.',
};

const saveForControl: Readonly<Record<SupportedLanguage, string>> = {
  fr: 'Enregistrer et renvoyer au contrôle',
  en: 'Save and send back for review',
};

function articleReserved(): ScriptedAnswer {
  return {
    status: 409,
    body: {
      type: 'urn:collector:probleme:article-reserve',
      title: 'Article réservé',
      status: 409,
    },
  };
}

async function openArticle(etat: ArticleState, language: SupportedLanguage): Promise<void> {
  givenArticles(storedArticle(1, { etat, revision: 4 }));
  await openAsSeller(`/vendeur/articles/${id}`, language);
  await editorHeading(title);
}

describe('US-031-AC5 front : modification refusée sur un article réservé', () => {
  it.each(languages)(
    'Given la langue %s et un article publié réservé, When le vendeur enregistre une modification refusée en 409 article-reserve, Then le refus est expliqué',
    async (language) => {
      await openArticle('publie', language);
      marketplace().scriptedWrites.push(articleReserved());
      await type(field(sellerWording[language].title), 'Prix cassé');

      await click(screen.getByRole('button', { name: saveForControl[language] }));

      expect(await screen.findByText(reservedMessage[language])).toBeInTheDocument();
      expect(writesTo('PUT', id)).toHaveLength(1);
    },
  );

  it.each(languages)(
    'Given la langue %s et le refus 409 article-reserve, When le vendeur regarde le formulaire, Then sa saisie est conservée',
    async (language) => {
      await openArticle('publie', language);
      marketplace().scriptedWrites.push(articleReserved());
      await type(field(sellerWording[language].title), 'Ma saisie à garder');
      await type(field(sellerWording[language].description), 'Description en cours');

      await click(screen.getByRole('button', { name: saveForControl[language] }));
      await screen.findByText(reservedMessage[language]);

      expect(field(sellerWording[language].title)).toHaveValue('Ma saisie à garder');
      expect(field(sellerWording[language].description)).toHaveValue('Description en cours');
    },
  );

  it('Given le refus 409 article-reserve, When il s’affiche, Then il est annoncé aux technologies d’assistance', async () => {
    await openArticle('publie', 'fr');
    marketplace().scriptedWrites.push(articleReserved());
    await type(field(sellerWording.fr.title), 'Prix cassé');

    await click(screen.getByRole('button', { name: saveForControl.fr }));
    await screen.findByText(reservedMessage.fr);

    expect(liveRegionsText()).toContain(reservedMessage.fr);
  });

  it('Given le refus 409 article-reserve, When le vendeur regarde l’écran, Then l’article reste présenté publié, rien n’est réécrit', async () => {
    await openArticle('publie', 'fr');
    marketplace().scriptedWrites.push(articleReserved());
    await type(field(sellerWording.fr.title), 'Prix cassé');

    await click(screen.getByRole('button', { name: saveForControl.fr }));
    await screen.findByText(reservedMessage.fr);

    expect(marketplace().articles.get(id)).toMatchObject({
      etat: 'publie',
      revision: 4,
      titre: title,
    });
    expect(screen.queryByText(sellerWording.fr.states.en_controle)).not.toBeInTheDocument();
  });

  it.each(languages)(
    'Given la langue %s et un brouillon dont la soumission est refusée en 409 article-reserve, When le vendeur soumet, Then le refus est expliqué',
    async (language) => {
      await openArticle('brouillon', language);
      marketplace().scriptedWrites.push(articleReserved());

      await click(screen.getByRole('button', { name: sellerWording[language].submit }));

      expect(await screen.findByText(reservedMessage[language])).toBeInTheDocument();
    },
  );
});
