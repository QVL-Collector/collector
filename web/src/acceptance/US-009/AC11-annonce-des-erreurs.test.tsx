import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, onTestFinished } from 'vitest';
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
} from '../US-013/sellerMarketplaceHarness';
import { categoryField, editorHeading, field, fr, sellerWording } from '../US-013/sellerScreens';

type FocusReceived = {
  readonly element: Element;
  readonly invalid: string | null;
  readonly description: string;
};

const languages: readonly SupportedLanguage[] = ['fr', 'en'];

const wordingOf: Readonly<
  Record<
    SupportedLanguage,
    { tooLong: string; invalidAmount: string; threeErrors: string; oneError: string }
  >
> = {
  fr: {
    tooLong: 'Texte trop long.',
    invalidAmount: 'Valeur invalide. Exemple de montant : 12,50.',
    threeErrors: 'Le formulaire contient 3 erreurs : corrigez les champs signalés.',
    oneError: 'Le formulaire contient 1 erreur : corrigez le champ signalé.',
  },
  en: {
    tooLong: 'Text too long.',
    invalidAmount: 'Invalid value. Amount example: 12.50.',
    threeErrors: 'The form contains 3 errors: fix the highlighted fields.',
    oneError: 'The form contains 1 error: fix the highlighted field.',
  },
};

const threeFieldErrors = [
  { champ: 'titre', code: 'trop_long' },
  { champ: 'categorieId', code: 'categorie_inconnue' },
  { champ: 'prix.montantCentimes', code: 'negatif' },
];

function describedText(element: Element): string {
  return (element.getAttribute('aria-describedby') ?? '')
    .split(/\s+/)
    .filter((id) => id !== '')
    .map((id) => document.getElementById(id)?.textContent ?? '')
    .join(' ');
}

function recordFocusReceived(): FocusReceived[] {
  const received: FocusReceived[] = [];
  const controller = new AbortController();
  document.addEventListener(
    'focusin',
    (event) => {
      if (event.target instanceof Element) {
        received.push({
          element: event.target,
          invalid: event.target.getAttribute('aria-invalid'),
          description: describedText(event.target),
        });
      }
    },
    { signal: controller.signal },
  );
  onTestFinished(() => {
    controller.abort();
  });
  return received;
}

function isInsideAlert(node: Node): boolean {
  const element = node instanceof Element ? node : node.parentElement;
  return element?.closest('[role="alert"]') != null;
}

function countAlertRenders(): () => number {
  let renders = 0;
  const observer = new MutationObserver((records) => {
    renders += records.filter((record) =>
      [record.target, ...Array.from(record.addedNodes)].some(isInsideAlert),
    ).length;
  });
  observer.observe(document.body, { subtree: true, childList: true, characterData: true });
  onTestFinished(() => {
    observer.disconnect();
  });
  return () => renders;
}

function focusReceivedBy(
  received: readonly FocusReceived[],
  element: Element,
): FocusReceived | undefined {
  return received.find((focus) => focus.element === element);
}

async function openDraftEditor(language: SupportedLanguage): Promise<void> {
  givenArticles(storedArticle(1));
  await openAsSeller(`/vendeur/articles/${articleId(1)}`, language);
  await editorHeading('Lot de cartes 1');
}

async function saveRejectedWithThreeErrors(language: SupportedLanguage): Promise<void> {
  await openDraftEditor(language);
  marketplace().scriptedWrites.push(invalidInput(threeFieldErrors));
  await type(field(sellerWording[language].title), 'Lot de cartes modifié');
  await click(screen.getByRole('button', { name: sellerWording[language].save }));
  await screen.findByRole('alert');
}

async function createWithUnreadablePrice(language: SupportedLanguage): Promise<void> {
  const wording = sellerWording[language];
  await openAsSeller('/vendeur/articles/nouveau', language);
  await editorHeading(wording.newItemTitle);
  await type(field(wording.title), 'Figurine en plomb');
  await type(field(wording.price), 'abc');
  await click(screen.getByRole('button', { name: wording.create }));
  await screen.findByRole('alert');
}

async function pressEnterIn(element: HTMLElement): Promise<void> {
  await userEvent.type(element, '{Enter}');
}

describe('US-009-AC11 le premier champ en erreur reçoit le focus déjà décrit par son erreur', () => {
  it.each(languages)(
    'Given une saisie refusée en 400 sur trois champs (%s), When le focus arrive sur le titre, Then le titre est déjà aria-invalid et décrit par son message d’erreur',
    async (language) => {
      await openDraftEditor(language);
      marketplace().scriptedWrites.push(invalidInput(threeFieldErrors));
      await type(field(sellerWording[language].title), 'Lot de cartes modifié');
      const received = recordFocusReceived();

      await click(screen.getByRole('button', { name: sellerWording[language].save }));

      await waitFor(() => {
        expect(field(sellerWording[language].title)).toHaveFocus();
      });
      const focus = focusReceivedBy(received, field(sellerWording[language].title));
      expect(focus?.invalid).toBe('true');
      expect(focus?.description).toContain(wordingOf[language].tooLong);
    },
  );

  it.each(languages)(
    'Given un prix illisible (%s), When le vendeur crée le brouillon, Then le focus arrive sur le prix déjà aria-invalid et décrit par son message d’erreur',
    async (language) => {
      const wording = sellerWording[language];
      await openAsSeller('/vendeur/articles/nouveau', language);
      await editorHeading(wording.newItemTitle);
      await type(field(wording.title), 'Figurine en plomb');
      await type(field(wording.price), 'abc');
      const received = recordFocusReceived();

      await click(screen.getByRole('button', { name: wording.create }));

      await waitFor(() => {
        expect(field(wording.price)).toHaveFocus();
      });
      const focus = focusReceivedBy(received, field(wording.price));
      expect(focus?.invalid).toBe('true');
      expect(focus?.description).toContain(wordingOf[language].invalidAmount);
    },
  );

  it('Given une erreur sur la seule catégorie, When le focus arrive sur la catégorie, Then elle est déjà aria-invalid et décrite par son message d’erreur', async () => {
    await openDraftEditor('fr');
    marketplace().scriptedWrites.push(
      invalidInput([{ champ: 'categorieId', code: 'categorie_inconnue' }]),
    );
    const received = recordFocusReceived();

    await click(screen.getByRole('button', { name: fr.save }));

    await waitFor(() => {
      expect(categoryField()).toHaveFocus();
    });
    const focus = focusReceivedBy(received, categoryField());
    expect(focus?.invalid).toBe('true');
    expect(focus?.description).toContain('Catégorie inconnue.');
  });
});

describe('US-009-AC11 une nouvelle soumission en erreur est de nouveau annoncée', () => {
  it.each(languages)(
    'Given trois erreurs affichées (%s), When le vendeur valide par Entrée dans le prix et l’API renvoie les mêmes trois erreurs, Then le résumé est de nouveau annoncé dans une seule alerte',
    async (language) => {
      await saveRejectedWithThreeErrors(language);
      marketplace().scriptedWrites.push(invalidInput(threeFieldErrors));
      const alertRenders = countAlertRenders();

      await pressEnterIn(field(sellerWording[language].price));

      await waitFor(() => {
        expect(alertRenders()).toBeGreaterThan(0);
      });
      expect(screen.getAllByRole('alert')).toHaveLength(1);
      expect(screen.getByRole('alert')).toHaveTextContent(wordingOf[language].threeErrors);
    },
  );

  it.each(languages)(
    'Given un prix illisible signalé (%s), When le vendeur valide de nouveau par Entrée dans le prix, Then le résumé à une erreur est de nouveau annoncé dans une seule alerte',
    async (language) => {
      await createWithUnreadablePrice(language);
      const alertRenders = countAlertRenders();

      await pressEnterIn(field(sellerWording[language].price));

      await waitFor(() => {
        expect(alertRenders()).toBeGreaterThan(0);
      });
      expect(screen.getAllByRole('alert')).toHaveLength(1);
      expect(screen.getByRole('alert')).toHaveTextContent(wordingOf[language].oneError);
    },
  );

  it('Given un prix illisible signalé, When le vendeur clique trois fois de suite sur Créer, Then chaque soumission est annoncée et une seule alerte subsiste', async () => {
    await createWithUnreadablePrice('fr');
    const alertRenders = countAlertRenders();

    for (const attempt of [1, 2, 3]) {
      await click(screen.getByRole('button', { name: fr.create }));
      await waitFor(() => {
        expect(alertRenders()).toBeGreaterThanOrEqual(attempt);
      });
    }

    expect(screen.getAllByRole('alert')).toHaveLength(1);
    expect(screen.getByRole('alert')).toHaveTextContent(wordingOf.fr.oneError);
  });

  it('Given trois erreurs affichées, When une nouvelle soumission par Entrée échoue de la même façon, Then le focus revient au premier champ en erreur, toujours décrit par son erreur', async () => {
    await saveRejectedWithThreeErrors('fr');
    marketplace().scriptedWrites.push(invalidInput(threeFieldErrors));

    await pressEnterIn(field(fr.price));

    await waitFor(() => {
      expect(field(fr.title)).toHaveFocus();
    });
    expect(field(fr.title)).toHaveAttribute('aria-invalid', 'true');
    expect(field(fr.title)).toHaveAccessibleDescription(
      expect.stringContaining(wordingOf.fr.tooLong) as string,
    );
  });

  it.each(languages)(
    'Given une seconde soumission en erreur annoncée (%s), When axe-core analyse le formulaire, Then aucune violation serious ou critical',
    async (language) => {
      await saveRejectedWithThreeErrors(language);
      marketplace().scriptedWrites.push(invalidInput(threeFieldErrors));
      await pressEnterIn(field(sellerWording[language].price));
      await waitFor(() => {
        expect(field(sellerWording[language].title)).toHaveFocus();
      });

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );
});
