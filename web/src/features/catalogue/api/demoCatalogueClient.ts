import { ApiError } from '../../../api/ApiError';
import { waitForDemoLatency } from '../../../api/demoLatency';
import type { DemoScenario } from '../../../config/runtimeConfig';
import { parseCategories } from '../../../shared/marketplacePayload';
import type { CatalogueClient, CatalogueQuery } from '../domain/catalogue';
import { unknownCategoryCode } from '../domain/catalogueProblems';
import { parseCataloguePage, parsePublishedArticle } from './cataloguePayload';
import { demoArticles, demoCategories, type DemoArticlePayload } from './demoArticles';

const httpStatuses = { badRequest: 400, notFound: 404, serviceUnavailable: 503 } as const;

const unknownCategoryProblem = {
  type: 'urn:collector:probleme:saisie-invalide',
  fieldProblems: [{ field: 'categorie', code: unknownCategoryCode }],
};

function scenarioArticles(scenario: DemoScenario): readonly DemoArticlePayload[] {
  return scenario === 'empty' ? [] : demoArticles;
}

function filterByCategory(
  articles: readonly DemoArticlePayload[],
  categoryId: string | null,
): readonly DemoArticlePayload[] {
  if (categoryId === null) {
    return articles;
  }
  if (!demoCategories.some((category) => category.id === categoryId)) {
    throw new ApiError(httpStatuses.badRequest, unknownCategoryProblem);
  }
  return articles.filter((article) => article.categorie.id === categoryId);
}

function toEntry({ id, titre, prix, categorie }: DemoArticlePayload): unknown {
  return { id, titre, prix, categorie };
}

function toPagePayload(
  articles: readonly DemoArticlePayload[],
  { page, pageSize }: CatalogueQuery,
): unknown {
  const start = (page - 1) * pageSize;
  return {
    elements: articles.slice(start, start + pageSize).map(toEntry),
    page,
    taille: pageSize,
    total: articles.length,
  };
}

async function answerAfterLatency(scenario: DemoScenario, signal?: AbortSignal): Promise<void> {
  await waitForDemoLatency(signal);
  if (scenario === 'error') {
    throw new ApiError(httpStatuses.serviceUnavailable);
  }
}

export function createDemoCatalogueClient(scenario: DemoScenario): CatalogueClient {
  return {
    async listPublishedArticles(query, signal) {
      await answerAfterLatency(scenario, signal);
      const articles = filterByCategory(scenarioArticles(scenario), query.categoryId);
      return parseCataloguePage(toPagePayload(articles, query));
    },
    async listCategories(signal) {
      await answerAfterLatency(scenario, signal);
      return parseCategories({ elements: demoCategories });
    },
    async getPublishedArticle(id, signal) {
      await answerAfterLatency(scenario, signal);
      const article = scenarioArticles(scenario).find((candidate) => candidate.id === id);
      if (article === undefined) {
        throw new ApiError(httpStatuses.notFound);
      }
      return parsePublishedArticle(article);
    },
  };
}
