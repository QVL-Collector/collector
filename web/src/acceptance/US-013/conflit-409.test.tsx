import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import {
  articleId,
  click,
  expectAnnounced,
  givenArticles,
  marketplace,
  notModifiable,
  notSubmittable,
  openAsSeller,
  storedArticle,
  type,
  waitAtLeast,
  writesTo,
} from './sellerMarketplaceHarness';
import { editorHeading, field, fr } from './sellerScreens';
import { expectStateShown, liveRegionsText, statePanel } from './sellerStatePanel';

const id = articleId(1);
const title = 'Lot de cartes 1';
const changedElsewhere = 'Lot modifié dans un autre onglet';
const lockedOnSave =
  'L’annonce n’est plus modifiable dans son état actuel. La page a été mise à jour.';
const lockedOnSubmit =
  'L’annonce ne peut pas être soumise dans son état actuel. Elle vient d’être rechargée.';

async function openDraft(): Promise<void> {
  givenArticles(storedArticle(1));
  await openAsSeller(`/vendeur/articles/${id}`);
  await editorHeading(title);
}

function changeElsewhere(overrides: Parameters<typeof storedArticle>[1]): void {
  const current = marketplace().articles.get(id);
  marketplace().articles.set(id, {
    ...storedArticle(1, overrides),
    revision: (current?.revision ?? 1) + 1,
  });
}

describe('US-013 front, conflit 409 de révision périmée à l’enregistrement', () => {
  it('Given l’article changé ailleurs, When le vendeur enregistre sa version, Then le conflit est expliqué et rien n’est écrasé', async () => {
    await openDraft();
    changeElsewhere({ titre: changedElsewhere });

    await type(field(fr.title), 'Ma version locale');
    await click(screen.getByRole('button', { name: fr.save }));

    expect(await screen.findByText('Cet article a été modifié entre-temps')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: fr.reload })).toBeInTheDocument();
    expect(marketplace().articles.get(id)?.titre).toBe(changedElsewhere);
    expect(
      writesTo('PUT', id).map((request) => (request.body as { revision: number }).revision),
    ).toEqual([1]);
  });

  it('Given le conflit expliqué, When le vendeur recharge la dernière version, Then le formulaire la montre et l’écriture suivante envoie la nouvelle révision', async () => {
    await openDraft();
    changeElsewhere({ titre: changedElsewhere });
    await type(field(fr.title), 'Ma version locale');
    await click(screen.getByRole('button', { name: fr.save }));

    await click(await screen.findByRole('button', { name: fr.reload }));

    await editorHeading(changedElsewhere);
    expect(field(fr.title)).toHaveValue(changedElsewhere);
    expect(screen.queryByText('Cet article a été modifié entre-temps')).not.toBeInTheDocument();
    await type(field(fr.title), 'Version finale');
    await click(screen.getByRole('button', { name: fr.save }));
    await expectAnnounced('Brouillon enregistré, révision 3.');
    expect((writesTo('PUT', id).at(-1)?.body as { revision: number }).revision).toBe(2);
  });

  it('Given un conflit, When le vendeur recharge par le clavier, Then le bouton « Recharger la dernière version » est atteignable et actionnable', async () => {
    await openDraft();
    changeElsewhere({ titre: changedElsewhere });
    await type(field(fr.title), 'Ma version locale');
    await click(screen.getByRole('button', { name: fr.save }));
    const reload = await screen.findByRole('button', { name: fr.reload });

    reload.focus();
    await userEvent.keyboard('{Enter}');

    await editorHeading(changedElsewhere);
  });
});

describe('US-013 front, conflit 409 de révision périmée à la soumission', () => {
  it('Given l’article changé ailleurs, When le vendeur le soumet, Then le conflit est expliqué, l’article est relu et rien n’est soumis', async () => {
    await openDraft();
    changeElsewhere({ titre: changedElsewhere });

    await click(screen.getByRole('button', { name: fr.submit }));

    expect(
      await screen.findByText(
        'L’article a changé depuis son affichage. Il vient d’être rechargé : vérifiez-le, puis soumettez-le de nouveau.',
      ),
    ).toBeInTheDocument();
    await editorHeading(changedElsewhere);
    expect(marketplace().articles.get(id)?.etat).toBe('brouillon');
  });
});

describe('US-013 front, 409 article plus modifiable ou plus soumettable', () => {
  it('Given l’article passé en contrôle ailleurs, When le vendeur enregistre, Then l’échec reste affiché et annoncé et l’article est relu', async () => {
    await openDraft();
    await type(field(fr.title), 'Ma version locale');
    changeElsewhere({ etat: 'en_controle' });
    marketplace().scriptedWrites.push(notModifiable());

    await click(screen.getByRole('button', { name: fr.save }));

    await expectStateShown(fr.states.en_controle);
    expect(await screen.findByText(lockedOnSave)).toBeInTheDocument();
    expect(liveRegionsText()).toContain(lockedOnSave);
    await waitAtLeast(2500);
    expect(screen.getByText(lockedOnSave)).toBeInTheDocument();
    expect(marketplace().articles.get(id)?.titre).toBe(title);
  });

  it('Given l’article publié ailleurs, When le vendeur le soumet, Then l’échec reste affiché dans le panneau d’état, annoncé, et l’article est relu', async () => {
    await openDraft();
    changeElsewhere({ etat: 'publie' });
    marketplace().scriptedWrites.push(notSubmittable());

    await click(screen.getByRole('button', { name: fr.submit }));

    await expectStateShown(fr.states.publie);
    expect(await within(statePanel()).findByText(lockedOnSubmit)).toBeInTheDocument();
    expect(liveRegionsText()).toContain(lockedOnSubmit);
    await waitAtLeast(2500);
    expect(within(statePanel()).getByText(lockedOnSubmit)).toBeInTheDocument();
  });
});
