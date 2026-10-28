import { screen, waitFor, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { currentPath } from '../US-008/identityProviderHarness';
import {
  articleId,
  chooseOption,
  click,
  expectAnnounced,
  givenArticles,
  marketplace,
  openAsSeller,
  readsOf,
  storedArticle,
  type,
  writesTo,
} from './sellerMarketplaceHarness';
import { categoryField, editorHeading, field, fr } from './sellerScreens';

const firstId = articleId(1);

describe('US-013 front, création d’un brouillon depuis l’espace vendeur', () => {
  it('Given le vendeur sans article, When il ouvre « Mes articles », Then la liste vide l’invite à créer un brouillon', async () => {
    await openAsSeller('/vendeur');

    expect(
      await screen.findByRole('heading', { name: 'Aucun article pour le moment' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: fr.newItem })).toHaveAttribute(
      'href',
      '/vendeur/articles/nouveau',
    );
    expect(writesTo('GET', '/vendeur/articles?page=1&taille=20')[0]?.authorization).toMatch(
      /^Bearer \S+$/,
    );
  });

  it('Given le formulaire de création, When le vendeur le remplit et crée le brouillon, Then l’API reçoit le contenu saisi et l’article s’ouvre en brouillon révision 1', async () => {
    await openAsSeller('/vendeur/articles/nouveau');
    await editorHeading(fr.newItemTitle);

    await type(field(fr.title), 'Pièce de 5 francs Semeuse');
    await type(field(fr.description), 'Argent, 1960, très bon état');
    await chooseOption(fr.category, 'Monnaies');
    await type(field(fr.price), '12,50');
    await type(field(fr.shippingCost), '3');
    await click(screen.getByRole('button', { name: fr.create }));

    await editorHeading('Pièce de 5 francs Semeuse');
    expect(writesTo('POST', '/vendeur/articles').map((request) => request.body)).toEqual([
      {
        titre: 'Pièce de 5 francs Semeuse',
        description: 'Argent, 1960, très bon état',
        categorieId: 'monnaies',
        prix: { montantCentimes: 1250, devise: 'EUR' },
        fraisDePort: { montantCentimes: 300, devise: 'EUR' },
      },
    ]);
    const [created] = [...marketplace().articles.values()];
    expect(created?.revision).toBe(1);
    expect(currentPath()).toBe(`/vendeur/articles/${created?.id ?? ''}`);
    expect(screen.getByText(fr.states.brouillon)).toBeInTheDocument();
  });

  it('Given un brouillon créé, When le vendeur revient à « Mes articles », Then il y figure à l’état brouillon', async () => {
    await openAsSeller('/vendeur/articles/nouveau');
    await editorHeading(fr.newItemTitle);
    await type(field(fr.title), 'Carte postale ancienne');
    await click(screen.getByRole('button', { name: fr.create }));
    await editorHeading('Carte postale ancienne');

    await click(screen.getByRole('link', { name: 'Retour à mes articles' }));

    const list = await screen.findByRole('list', { name: fr.myItems });
    const card = within(list).getByRole('article');
    expect(within(card).getByText('Carte postale ancienne')).toBeInTheDocument();
    expect(within(card).getByText(fr.states.brouillon)).toBeInTheDocument();
  });

  it('Given le formulaire de création, When les catégories sont lues, Then seules celles de GET /categories sont proposées', async () => {
    marketplace().categories = [
      { id: 'jouets-anciens', libelle: 'Jouets anciens' },
      { id: 'vinyles', libelle: 'Vinyles' },
    ];
    await openAsSeller('/vendeur/articles/nouveau');
    await editorHeading(fr.newItemTitle);

    await waitFor(() => {
      expect(
        Array.from(categoryField().options)
          .map((option) => option.value)
          .filter((value) => value !== ''),
      ).toEqual(['jouets-anciens', 'vinyles']);
    });
    expect(writesTo('GET', '/marche/v1/categories').length).toBeGreaterThan(0);
  });

  it('Given les catégories indisponibles, When le formulaire s’ouvre, Then le vendeur en est informé', async () => {
    marketplace().categoriesAvailable = false;
    await openAsSeller('/vendeur/articles/nouveau');
    await editorHeading(fr.newItemTitle);

    expect(
      await screen.findByText('Les catégories sont momentanément indisponibles.', undefined, {
        timeout: 10000,
      }),
    ).toBeInTheDocument();
  }, 15000);
});

describe('US-013 front, modification d’un brouillon', () => {
  it('Given un brouillon en révision 3, When le vendeur le modifie et l’enregistre, Then la révision lue est envoyée et la nouvelle révision est annoncée', async () => {
    givenArticles(storedArticle(1, { revision: 3 }));
    await openAsSeller(`/vendeur/articles/${firstId}`);
    await editorHeading('Lot de cartes 1');

    await type(field(fr.title), 'Lot de cartes rares');
    await click(screen.getByRole('button', { name: fr.save }));

    await expectAnnounced('Brouillon enregistré, révision 4.');
    expect(writesTo('PUT', firstId).map((request) => request.body)).toEqual([
      {
        revision: 3,
        titre: 'Lot de cartes rares',
        description: 'Cartes anciennes en bon état',
        categorieId: 'cartes',
        prix: { montantCentimes: 1250, devise: 'EUR' },
        fraisDePort: { montantCentimes: 300, devise: 'EUR' },
      },
    ]);
    expect(marketplace().articles.get(firstId)?.titre).toBe('Lot de cartes rares');
  });

  it('Given un brouillon modifié deux fois de suite, When il est enregistré à chaque fois, Then chaque écriture envoie la dernière révision connue', async () => {
    givenArticles(storedArticle(1));
    await openAsSeller(`/vendeur/articles/${firstId}`);
    await editorHeading('Lot de cartes 1');

    await type(field(fr.title), 'Première version');
    await click(screen.getByRole('button', { name: fr.save }));
    await expectAnnounced('Brouillon enregistré, révision 2.');
    await type(field(fr.title), 'Seconde version');
    await click(screen.getByRole('button', { name: fr.save }));
    await expectAnnounced('Brouillon enregistré, révision 3.');

    expect(
      writesTo('PUT', firstId).map((request) => (request.body as { revision: number }).revision),
    ).toEqual([1, 2]);
  });

  it('Given un brouillon dont le prix est vidé, When il est enregistré, Then le prix part vide sans bloquer l’enregistrement', async () => {
    givenArticles(storedArticle(1));
    await openAsSeller(`/vendeur/articles/${firstId}`);
    await editorHeading('Lot de cartes 1');

    await type(field(fr.price), '');
    await click(screen.getByRole('button', { name: fr.save }));

    await expectAnnounced('Brouillon enregistré, révision 2.');
    expect((writesTo('PUT', firstId)[0]?.body as { prix?: unknown }).prix ?? null).toBeNull();
  });

  it('Given un article inexistant ou d’un autre vendeur, When le vendeur ouvre son adresse, Then « Article introuvable » s’affiche', async () => {
    await openAsSeller(`/vendeur/articles/${articleId(999)}`);

    expect(await screen.findByText('Article introuvable')).toBeInTheDocument();
    expect(readsOf(articleId(999)).length).toBeGreaterThan(0);
  });
});
