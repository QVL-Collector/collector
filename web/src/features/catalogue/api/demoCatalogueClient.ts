import { ApiError } from '../../../api/ApiError';
import { waitForDemoLatency } from '../../../api/demoLatency';
import type { DemoScenario } from '../../../config/runtimeConfig';
import type { CatalogueClient, CatalogueQuery } from '../domain/catalogue';
import { parseCataloguePage } from './cataloguePayload';
import { demoArticles } from './demoArticles';

const serviceUnavailableStatus = 503;

function scenarioArticles(scenario: DemoScenario): readonly unknown[] {
  return scenario === 'empty' ? [] : demoArticles;
}

function toPayload(articles: readonly unknown[], { page, pageSize }: CatalogueQuery): unknown {
  const start = (page - 1) * pageSize;
  return {
    elements: articles.slice(start, start + pageSize),
    page,
    taille: pageSize,
    total: articles.length,
  };
}

export function createDemoCatalogueClient(scenario: DemoScenario): CatalogueClient {
  return {
    async listPublishedArticles(query, signal) {
      await waitForDemoLatency(signal);
      if (scenario === 'error') {
        throw new ApiError(serviceUnavailableStatus);
      }
      return parseCataloguePage(toPayload(scenarioArticles(scenario), query));
    },
  };
}
