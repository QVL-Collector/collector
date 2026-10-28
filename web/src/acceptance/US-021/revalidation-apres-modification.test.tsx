import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  articleId,
  axeTimeout,
  blockingViolations,
  click,
  expectAnnounced,
  givenArticles,
  marketplace,
  openAsSeller,
  storedArticle,
  type,
  writesTo,
  type ArticleState,
  type ControlReason,
} from '../US-013/sellerMarketplaceHarness';
import { editorHeading, field, fr } from '../US-013/sellerScreens';
import { expectStateShown } from '../US-013/sellerStatePanel';

const id = articleId(1);
const title = 'Lot de cartes 1';
const editedTitle = 'Lot de cartes 1, appelez-moi';
const removalWarning = 'L’annonce sera retirée du catalogue';
const restartWarning = 'Le contrôle repartira sur la nouvelle version';
const saveForControl = 'Enregistrer et renvoyer au contrôle';

const reasons: Readonly<Record<ArticleState, ControlReason | null>> = {
  brouillon: null,
  en_controle: null,
  publie: null,
  refuse: 'annonce_incomplete',
  a_verifier: 'categorie_inconnue',
};

async function openArticle(etat: ArticleState, revision = 3): Promise<void> {
  givenArticles(storedArticle(1, { etat, motif: reasons[etat], revision }));
  await openAsSeller(`/vendeur/articles/${id}`);
  await editorHeading(title);
  await expectStateShown(fr.states[etat]);
}

async function editTitle(): Promise<void> {
  await type(field(fr.title), editedTitle);
}

describe('US-021 AC4 avertissement avant de modifier un article publié', () => {
  it('Given un article publié, When le vendeur le modifie, Then il est averti que l’annonce sera retirée du catalogue le temps du contrôle', async () => {
    await openArticle('publie');

    await editTitle();

    expect(screen.getByText(removalWarning)).toBeInTheDocument();
    expect(
      screen.getByText(/retire l’annonce du catalogue le temps du contrôle/),
    ).toBeInTheDocument();
    expect(writesTo('PUT')).toEqual([]);
  });

  it.each(['brouillon', 'refuse'] as const)(
    'Given un article %s, When le vendeur le modifie, Then aucun avertissement de retrait du catalogue ni de reprise du contrôle',
    async (etat) => {
      await openArticle(etat);

      await editTitle();

      expect(screen.queryByText(removalWarning)).not.toBeInTheDocument();
      expect(screen.queryByText(restartWarning)).not.toBeInTheDocument();
    },
  );

  it.each(['en_controle', 'a_verifier'] as const)(
    'Given un article %s, When le vendeur le modifie, Then il est averti que le contrôle repartira sur la nouvelle version',
    async (etat) => {
      await openArticle(etat);

      await editTitle();

      expect(screen.getByText(restartWarning)).toBeInTheDocument();
      expect(screen.queryByText(removalWarning)).not.toBeInTheDocument();
    },
  );
});

describe('US-021 AC1 AC2 front, enregistrement d’une modification renvoyée au contrôle', () => {
  it('Given un article publié, When le vendeur enregistre le seul titre modifié, Then la révision lue est envoyée et l’état affiché devient « En contrôle »', async () => {
    await openArticle('publie', 4);
    await editTitle();

    await click(screen.getByRole('button', { name: saveForControl }));

    await expectStateShown(fr.states.en_controle);
    await expectAnnounced(fr.states.en_controle);
    expect(writesTo('PUT', id).map((request) => request.body)).toEqual([
      expect.objectContaining({ revision: 4, titre: editedTitle }),
    ]);
    expect(writesTo('POST', `${id}/soumission`)).toEqual([]);
    await editorHeading(editedTitle);
    expect(screen.queryByText(fr.states.publie)).not.toBeInTheDocument();
  });

  it.each(['en_controle', 'a_verifier'] as const)(
    'Given un article %s, When le vendeur enregistre une modification, Then l’état affiché est « En contrôle » sans motif',
    async (etat) => {
      await openArticle(etat, 5);
      await editTitle();

      await click(screen.getByRole('button', { name: saveForControl }));

      await editorHeading(editedTitle);
      await expectStateShown(fr.states.en_controle);
      expect(screen.queryByText(fr.unknownCategory)).not.toBeInTheDocument();
      expect(marketplace().articles.get(id)).toMatchObject({ etat: 'en_controle', revision: 6 });
    },
  );
});

describe('US-021 axe-core sur l’écran de modification', () => {
  it.each(['publie', 'en_controle', 'a_verifier'] as const)(
    'Given l’écran de modification d’un article %s avec l’avertissement affiché, When axe-core l’analyse, Then aucune violation serious ou critical',
    async (etat) => {
      await openArticle(etat);
      await editTitle();

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );
});
