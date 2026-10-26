import '@testing-library/jest-dom/vitest';
import { cleanup, render, type RenderResult } from '@testing-library/react';
import i18n from 'i18next';
import { afterEach, beforeAll, vi } from 'vitest';
import { App } from '../../app/App';
import type { AppServices } from '../../app/createAppServices';
import type {
  CatalogueArticle,
  CatalogueClient,
  CataloguePage,
  CatalogueQuery,
} from '../../features/catalogue/domain/catalogue';
import { initI18n } from '../../i18n/initI18n';
import type { SupportedLanguage } from '../../i18n/languages';

type ListPublishedArticles = CatalogueClient['listPublishedArticles'];

export type FakeCatalogue = {
  readonly client: CatalogueClient;
  readonly listPublishedArticles: ReturnType<typeof vi.fn<ListPublishedArticles>>;
};

function stubMatchMedia(): void {
  Object.defineProperty(window, 'matchMedia', {
    configurable: true,
    writable: true,
    value: (query: string): MediaQueryList => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => undefined,
      removeListener: () => undefined,
      addEventListener: () => undefined,
      removeEventListener: () => undefined,
      dispatchEvent: () => false,
    }),
  });
}

class RequestWithoutJsdomSignal extends Request {
  constructor(input: RequestInfo | URL, init?: RequestInit) {
    super(input, init === undefined ? undefined : { ...init, signal: null });
  }
}

beforeAll(async () => {
  stubMatchMedia();
  globalThis.Request = RequestWithoutJsdomSignal;
  await initI18n('fr');
});

afterEach(async () => {
  cleanup();
  vi.unstubAllGlobals();
  await i18n.changeLanguage('fr');
  window.history.replaceState(null, '', '/');
});

export async function useLanguage(language: SupportedLanguage): Promise<void> {
  await i18n.changeLanguage(language);
}

export function publishedArticle(
  index: number,
  title = `Objet de collection ${String(index)}`,
): CatalogueArticle {
  return {
    id: `article-${String(index)}`,
    title,
    price: { amountInCents: 1000 + index * 125, currency: 'EUR' },
    category: { id: `categorie-${String(index % 3)}`, label: `Catégorie ${String(index % 3)}` },
  };
}

export function publishedArticles(count: number): CatalogueArticle[] {
  return Array.from({ length: count }, (_, position) => publishedArticle(position + 1));
}

export function pageOf(
  articles: readonly CatalogueArticle[],
  query: CatalogueQuery,
): CataloguePage {
  const start = (query.page - 1) * query.pageSize;
  return {
    articles: articles.slice(start, start + query.pageSize),
    page: query.page,
    pageSize: query.pageSize,
    totalItems: articles.length,
  };
}

export function fakeCatalogue(implementation: ListPublishedArticles): FakeCatalogue {
  const listPublishedArticles = vi.fn<ListPublishedArticles>(implementation);
  return { client: { listPublishedArticles }, listPublishedArticles };
}

export function catalogueContaining(articles: readonly CatalogueArticle[]): FakeCatalogue {
  return fakeCatalogue((query) => Promise.resolve(pageOf(articles, query)));
}

export function catalogueNeverAnswering(): FakeCatalogue {
  return fakeCatalogue(() => new Promise<CataloguePage>(() => undefined));
}

export function catalogueFailing(): FakeCatalogue {
  return fakeCatalogue(() => Promise.reject(new Error('indisponible')));
}

export function renderApp(services: AppServices, url = '/'): RenderResult {
  window.history.replaceState(null, '', url);
  return render(<App services={services} />);
}

export function renderCatalogue(catalogue: FakeCatalogue, url = '/'): RenderResult {
  return renderApp({ catalogueClient: catalogue.client }, url);
}

export function formattedPrice(amountInCents: number, locale: string): string {
  return new Intl.NumberFormat(locale, { style: 'currency', currency: 'EUR' })
    .format(amountInCents / 100)
    .replace(/\s+/g, ' ');
}

export function currentSearch(): URLSearchParams {
  return new URLSearchParams(window.location.search);
}
