import { screen, within } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { createAppServices } from '../../app/createAppServices';
import { renderApp } from './catalogueHarness';

const publicApiConfig = {
  marketplaceApiUrl: '/marche',
  catalogue: { source: 'api', demoScenario: 'nominal' },
} as const;

const publishedPayload = {
  elements: [
    {
      id: 'a1',
      titre: 'Montre de gousset 1920',
      prix: { montantCentimes: 15000, devise: 'EUR' },
      categorie: { id: 'c1', libelle: 'Horlogerie' },
    },
    {
      id: 'a2',
      titre: 'Carte Pokémon Dracaufeu',
      prix: { montantCentimes: 4999, devise: 'EUR' },
      categorie: { id: 'c2', libelle: 'Cartes' },
    },
  ],
  page: 1,
  taille: 12,
  total: 2,
};

function jsonResponse(body: unknown, status: number, contentType: string): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': contentType } });
}

function stubFetch(response: () => Response): ReturnType<typeof vi.fn<typeof fetch>> {
  const fetchSpy = vi.fn<typeof fetch>(() => Promise.resolve(response()));
  vi.stubGlobal('fetch', fetchSpy);
  return fetchSpy;
}

function requestOf(fetchSpy: ReturnType<typeof vi.fn<typeof fetch>>): Request {
  const call = fetchSpy.mock.calls[0];
  if (call === undefined) {
    throw new Error('Aucune requête émise');
  }
  const [input, init] = call;
  return new Request(
    new URL(input instanceof Request ? input.url : input.toString(), window.location.origin),
    init,
  );
}

async function renderWithPublicApi(): Promise<void> {
  renderApp(await createAppServices(publicApiConfig));
}

describe('US-010-AC2 le front n’expose que les articles publiés de l’API publique', () => {
  it('Given un visiteur non connecté, When le catalogue s’ouvre, Then la liste est demandée à l’API publique sans authentification', async () => {
    const fetchSpy = stubFetch(() => jsonResponse(publishedPayload, 200, 'application/json'));
    await renderWithPublicApi();

    await screen.findByRole('list', { name: 'Articles publiés' });

    const request = requestOf(fetchSpy);
    const url = new URL(request.url);
    expect(request.method).toBe('GET');
    expect(url.pathname).toBe('/marche/v1/articles');
    expect(url.searchParams.get('page')).toBe('1');
    expect(request.headers.has('Authorization')).toBe(false);
  });

  it('Given l’API publique renvoie deux articles publiés, When le catalogue s’affiche, Then exactement ces deux articles sont listés', async () => {
    stubFetch(() => jsonResponse(publishedPayload, 200, 'application/json'));
    await renderWithPublicApi();

    const items = within(
      await screen.findByRole('list', { name: 'Articles publiés' }),
    ).getAllByRole('listitem');

    expect(items).toHaveLength(2);
    expect(items[0]).toHaveTextContent('Montre de gousset 1920');
    expect(items[0]).toHaveTextContent('Horlogerie');
    expect(items[1]).toHaveTextContent('Carte Pokémon Dracaufeu');
  });

  it('Given l’API publique répond 404 en problem+json, When le catalogue s’ouvre, Then l’état d’erreur s’affiche sans article', async () => {
    stubFetch(() =>
      jsonResponse(
        { type: 'about:blank', title: 'Not Found', status: 404 },
        404,
        'application/problem+json',
      ),
    );
    await renderWithPublicApi();

    expect(
      await screen.findByRole(
        'heading',
        { name: 'Le catalogue est momentanément indisponible' },
        { timeout: 15000 },
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole('list', { name: 'Articles publiés' })).not.toBeInTheDocument();
  }, 20000);

  it('Given une réponse hors contrat, When le catalogue s’ouvre, Then rien n’est affiché comme article et l’état d’erreur apparaît', async () => {
    stubFetch(() =>
      jsonResponse(
        { elements: [{ id: 'x', titre: 42 }], page: 1, taille: 12, total: 1 },
        200,
        'application/json',
      ),
    );
    await renderWithPublicApi();

    expect(
      await screen.findByRole(
        'heading',
        { name: 'Le catalogue est momentanément indisponible' },
        { timeout: 15000 },
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole('list', { name: 'Articles publiés' })).not.toBeInTheDocument();
  }, 20000);
});
