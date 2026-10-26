import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it } from 'vitest';
import { createAppServices } from '../../app/createAppServices';
import {
  catalogueContaining,
  publishedArticle,
  publishedArticles,
  renderApp,
  renderCatalogue,
} from './catalogueHarness';

type PwnedWindow = Window & { pwned?: boolean };

const scriptTitle = '<script>window.pwned = true</script>Vase Gallé';
const imageTitle = '<img src="x" onerror="window.pwned = true">Timbre rare';
const markupPattern = /<script|<img/i;

afterEach(() => {
  delete (window as PwnedWindow).pwned;
});

async function findArticleList(): Promise<HTMLElement> {
  return screen.findByRole('list', { name: 'Articles publiés' });
}

describe('US-010-AC4 un titre contenant du balisage s’affiche comme du texte', () => {
  it('Given un titre avec <script>, When il est affiché, Then le texte brut est visible et aucun script n’est créé ni exécuté', async () => {
    renderCatalogue(catalogueContaining([publishedArticle(1, scriptTitle)]));

    const list = await findArticleList();

    expect(within(list).getByText(scriptTitle)).toBeInTheDocument();
    expect(list.querySelector('script')).toBeNull();
    expect((window as PwnedWindow).pwned).toBeUndefined();
  });

  it('Given un titre avec <img onerror>, When il est affiché, Then le texte brut est visible et aucune image ni gestionnaire n’est injecté', async () => {
    renderCatalogue(catalogueContaining([publishedArticle(1, imageTitle)]));

    const list = await findArticleList();

    expect(within(list).getByText(imageTitle)).toBeInTheDocument();
    expect(list.querySelector('img')).toBeNull();
    expect(list.querySelector('[onerror]')).toBeNull();
    expect((window as PwnedWindow).pwned).toBeUndefined();
  });

  it('Given des titres piégés parmi des titres ordinaires, When la liste s’affiche, Then le titre piégé sert aussi de libellé textuel accessible', async () => {
    renderCatalogue(
      catalogueContaining([publishedArticle(1, imageTitle), ...publishedArticles(3).slice(1)]),
    );

    const list = await findArticleList();

    expect(within(list).getByRole('heading', { name: imageTitle })).toBeInTheDocument();
    expect(within(list).getAllByRole('listitem')).toHaveLength(3);
  });

  it('Given le jeu de démonstration, When le visiteur parcourt toutes ses pages, Then les titres à balisage restent du texte sans élément injecté', async () => {
    const user = userEvent.setup();
    renderApp(
      await createAppServices({
        marketplaceApiUrl: '/marche',
        catalogue: { source: 'demo', demoScenario: 'nominal' },
      }),
    );
    const markupTitles: string[] = [];

    for (;;) {
      const list = await findArticleList();
      expect(list.querySelector('script, img, [onerror], iframe')).toBeNull();
      markupTitles.push(
        ...within(list)
          .getAllByRole('heading')
          .map((heading) => heading.textContent)
          .filter((title) => markupPattern.test(title)),
      );
      const next = screen.queryByRole('button', { name: 'Page suivante' });
      if (next === null || next.hasAttribute('disabled')) {
        break;
      }
      const statusBefore = screen.getByRole('status').textContent;
      await user.click(next);
      await screen.findByText(
        (_, element) =>
          element?.getAttribute('role') === 'status' && element.textContent !== statusBefore,
      );
    }

    expect(markupTitles.length).toBeGreaterThanOrEqual(2);
    expect((window as PwnedWindow).pwned).toBeUndefined();
  }, 30000);
});
