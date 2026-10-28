import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { switchLanguage } from '../US-008/identityProviderHarness';
import {
  articleId,
  click,
  concludeControl,
  expectAnnounced,
  givenArticles,
  marketplace,
  openAsSeller,
  readsOf,
  storedArticle,
  type,
  waitAtLeast,
  writesTo,
} from './sellerMarketplaceHarness';
import { editorHeading, field, fr, sellerWording } from './sellerScreens';
import { expectStateShown, statePanel, submitButtonIsUsable } from './sellerStatePanel';

const id = articleId(1);
const en = sellerWording.en;

function articleWithoutCreationDate(): Record<string, unknown> {
  return Object.fromEntries(Object.entries(storedArticle(1)).filter(([key]) => key !== 'creeLe'));
}
const title = 'Lot de cartes 1';

async function openDraft(overrides: Parameters<typeof storedArticle>[1] = {}): Promise<void> {
  givenArticles(storedArticle(1, overrides));
  await openAsSeller(`/vendeur/articles/${id}`);
  await editorHeading(title);
}

async function submitForControl(): Promise<void> {
  await click(screen.getByRole('button', { name: fr.submit }));
  await expectStateShown(fr.states.en_controle);
}

describe('US-013 AC1 front, soumission d’un brouillon complet', () => {
  it('Given un brouillon complet, When le vendeur le soumet, Then la révision lue est envoyée et l’état « En contrôle » est affiché et annoncé', async () => {
    await openDraft({ revision: 2 });

    await submitForControl();

    expect(writesTo('POST', `${id}/soumission`).map((request) => request.body)).toEqual([
      { revision: 2 },
    ]);
    await expectAnnounced(`« ${title} » : ${fr.states.en_controle}.`);
  });

  it('Given un article en contrôle, When le contrôle le publie, Then l’écran passe à « Publié » sans action du vendeur et l’annonce', async () => {
    await openDraft();
    await submitForControl();

    concludeControl(id, 'publie', null);

    await expectStateShown(fr.states.publie, 'fr', 6000);
    await expectAnnounced(`« ${title} » : ${fr.states.publie}.`);
    expect(screen.getByText('L’annonce est visible dans le catalogue.')).toBeInTheDocument();
  });

  it('Given un article en contrôle, When le temps passe, Then l’article est relu toutes les 2 secondes environ', async () => {
    await openDraft();
    await submitForControl();
    const readsBefore = readsOf(id).length;

    await waitAtLeast(4500);

    const pollReads = readsOf(id).length - readsBefore;
    expect(pollReads).toBeGreaterThanOrEqual(2);
    expect(pollReads).toBeLessThanOrEqual(6);
  });

  it('Given un article en contrôle puis publié, When l’écran l’affiche, Then « Soumettre au contrôle » n’est pas utilisable', async () => {
    await openDraft();
    await submitForControl();

    expect(submitButtonIsUsable(fr.submit)).toBe(false);
    concludeControl(id, 'publie', null);
    await expectStateShown(fr.states.publie, 'fr', 6000);
    expect(submitButtonIsUsable(fr.submit)).toBe(false);
  });

  it('Given chaque état, When il est affiché, Then il porte une icône décorative et un texte', async () => {
    await openDraft();

    const chip = within(statePanel()).getByText(fr.states.brouillon).parentElement;
    expect(chip?.querySelector('svg[aria-hidden="true"]')).not.toBeNull();
  });
});

describe('US-013 front, soumission bloquée tant que des modifications ne sont pas enregistrées', () => {
  it('Given un brouillon modifié non enregistré, When l’écran l’affiche, Then « Soumettre au contrôle » est désactivé et la raison est donnée', async () => {
    await openDraft();
    expect(submitButtonIsUsable(fr.submit)).toBe(true);

    await type(field(fr.title), 'Titre retouché');

    expect(submitButtonIsUsable(fr.submit)).toBe(false);
    expect(
      screen.getByText('Enregistrez vos modifications avant de soumettre l’annonce.'),
    ).toBeInTheDocument();
  });

  it('Given des modifications enregistrées, When l’enregistrement aboutit, Then la soumission redevient possible avec la nouvelle révision', async () => {
    await openDraft();
    await type(field(fr.title), 'Titre retouché');
    await click(screen.getByRole('button', { name: fr.save }));
    await expectAnnounced('Brouillon enregistré, révision 2.');

    expect(submitButtonIsUsable(fr.submit)).toBe(true);
    await submitForControl();
    expect(writesTo('POST', `${id}/soumission`)[0]?.body).toEqual({ revision: 2 });
  });

  it('Given des modifications non enregistrées, When le vendeur tente de soumettre, Then aucune soumission ne part', async () => {
    await openDraft();
    await type(field(fr.title), 'Autre');
    await click(screen.getByRole('button', { name: fr.submit })).catch(() => undefined);

    expect(writesTo('POST', `${id}/soumission`)).toEqual([]);
  });
});

describe('US-013 AC2 front, brouillon sans description refusé', () => {
  it('Given un article soumis sans description, When le contrôle le refuse, Then « Refusé » et le motif « Annonce incomplète » sont affichés et annoncés', async () => {
    await openDraft({ description: '' });
    await submitForControl();

    concludeControl(id, 'refuse', 'annonce_incomplete');

    await expectStateShown(fr.states.refuse, 'fr', 6000);
    expect(within(statePanel()).getByText(`Motif : ${fr.incomplete}`)).toBeInTheDocument();
    await expectAnnounced(`« ${title} » : ${fr.states.refuse}, motif : ${fr.incomplete}.`);
  });

  it('Given un article refusé pour annonce incomplète, When le vendeur ouvre « Mes articles », Then l’état et le motif y sont visibles', async () => {
    givenArticles(
      storedArticle(1, { etat: 'refuse', motif: 'annonce_incomplete', description: '' }),
    );
    await openAsSeller('/vendeur');

    const list = await screen.findByRole('list', { name: fr.myItems });
    expect(within(list).getByText(fr.states.refuse)).toBeInTheDocument();
    expect(within(list).getByText(`Motif : ${fr.incomplete}`)).toBeInTheDocument();
  });

  it('Given un article refusé affiché en anglais, When l’écran l’affiche, Then l’état et le motif sont traduits', async () => {
    givenArticles(
      storedArticle(1, { etat: 'refuse', motif: 'annonce_incomplete', description: '' }),
    );
    await openAsSeller(`/vendeur/articles/${id}`, 'en');
    await editorHeading(title);

    await expectStateShown(en.states.refuse, 'en');
    expect(within(statePanel('en')).getByText(`Reason: ${en.incomplete}`)).toBeInTheDocument();
  });

  it('Given un article refusé affiché en français, When le vendeur passe à l’anglais, Then le motif suit la langue', async () => {
    givenArticles(
      storedArticle(1, { etat: 'refuse', motif: 'annonce_incomplete', description: '' }),
    );
    await openAsSeller(`/vendeur/articles/${id}`);
    await editorHeading(title);

    await switchLanguage('en');

    await expectStateShown(en.states.refuse, 'en');
    expect(within(statePanel('en')).getByText(`Reason: ${en.incomplete}`)).toBeInTheDocument();
  });

  it('Given un refus annoncé en anglais, When le contrôle conclut, Then l’annonce est en anglais', async () => {
    givenArticles(storedArticle(1, { description: '' }));
    await openAsSeller(`/vendeur/articles/${id}`, 'en');
    await editorHeading(title);
    await click(screen.getByRole('button', { name: en.submit }));
    await expectStateShown(en.states.en_controle, 'en');

    concludeControl(id, 'refuse', 'annonce_incomplete');

    await expectAnnounced(`“${title}”: ${en.states.refuse}, reason: ${en.incomplete}.`, 6000);
  });
});

describe('US-013 AC3 front, article refusé corrigé puis resoumis', () => {
  it('Given un article refusé, When le vendeur le corrige, l’enregistre et le soumet de nouveau, Then il repasse en contrôle puis publié', async () => {
    await openDraft({ etat: 'refuse', motif: 'annonce_incomplete', description: '', revision: 3 });
    expect(within(statePanel()).getByText(`Motif : ${fr.incomplete}`)).toBeInTheDocument();

    await type(field(fr.description), 'Cartes anciennes complètes');
    await click(screen.getByRole('button', { name: fr.save }));
    await expectAnnounced('Brouillon enregistré, révision 4.');
    await submitForControl();

    expect(writesTo('POST', `${id}/soumission`)[0]?.body).toEqual({ revision: 4 });
    expect(within(statePanel()).queryByText(`Motif : ${fr.incomplete}`)).not.toBeInTheDocument();
    concludeControl(id, 'publie', null);
    await expectStateShown(fr.states.publie, 'fr', 6000);
  });
});

describe('US-013 AC4 front, catégorie inconnue du Contrôle', () => {
  it('Given un article soumis, When le contrôle conclut « à vérifier » pour catégorie inconnue, Then l’écran affiche « À vérifier » avec le motif et jamais « Publié »', async () => {
    await openDraft();
    await submitForControl();

    concludeControl(id, 'a_verifier', 'categorie_inconnue');

    await expectStateShown(fr.states.a_verifier, 'fr', 6000);
    expect(within(statePanel()).getByText(`Motif : ${fr.unknownCategory}`)).toBeInTheDocument();
    expect(within(statePanel()).queryByText(fr.states.publie)).not.toBeInTheDocument();
    await expectAnnounced(`« ${title} » : ${fr.states.a_verifier}, motif : ${fr.unknownCategory}.`);
    expect(submitButtonIsUsable(fr.submit)).toBe(false);
  });

  it('Given un article à vérifier en anglais, When l’écran l’affiche, Then état et motif sont traduits', async () => {
    givenArticles(storedArticle(1, { etat: 'a_verifier', motif: 'categorie_inconnue' }));
    await openAsSeller(`/vendeur/articles/${id}`, 'en');
    await editorHeading(title);

    await expectStateShown(en.states.a_verifier, 'en');
    expect(within(statePanel('en')).getByText(`Reason: ${en.unknownCategory}`)).toBeInTheDocument();
  });
});

describe('US-013 front, réponses hors contrat', () => {
  it('Given une lecture d’article sans date de création, When l’écran s’ouvre, Then l’état d’erreur s’affiche au lieu de l’article', async () => {
    const withoutCreation = articleWithoutCreationDate();
    givenArticles(storedArticle(1));
    marketplace().scriptedReads.push(
      ...Array.from({ length: 10 }, () => ({ status: 200, body: withoutCreation })),
    );
    await openAsSeller(`/vendeur/articles/${id}`);

    expect(
      await screen.findByText('L’article est momentanément indisponible', undefined, {
        timeout: 15000,
      }),
    ).toBeInTheDocument();
    expect(screen.queryByRole('heading', { level: 1, name: title })).not.toBeInTheDocument();
  });

  it('Given une lecture d’article à l’identifiant mal formé, When l’écran s’ouvre, Then l’état d’erreur s’affiche', async () => {
    givenArticles(storedArticle(1));
    marketplace().scriptedReads.push(
      ...Array.from({ length: 10 }, () => ({
        status: 200,
        body: { ...storedArticle(1), id: 'pas-un-identifiant' },
      })),
    );
    await openAsSeller(`/vendeur/articles/${id}`);

    expect(
      await screen.findByText('L’article est momentanément indisponible', undefined, {
        timeout: 15000,
      }),
    ).toBeInTheDocument();
  });

  it('Given une liste dont un article n’a pas de date de création, When « Mes articles » s’ouvre, Then l’état d’erreur s’affiche', async () => {
    const withoutCreation = articleWithoutCreationDate();
    marketplace().scriptedReads.push(
      ...Array.from({ length: 10 }, () => ({
        status: 200,
        body: { elements: [withoutCreation], page: 1, taille: 20, total: 1 },
      })),
    );
    await openAsSeller('/vendeur');

    expect(
      await screen.findByText('Vos articles sont momentanément indisponibles', undefined, {
        timeout: 15000,
      }),
    ).toBeInTheDocument();
    expect(screen.queryByText(title)).not.toBeInTheDocument();
  });
});
