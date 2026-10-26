import type { HttpClient } from '../../../api/httpClient';
import type { CatalogueConfig } from '../../../config/runtimeConfig';
import type { CatalogueClient } from '../domain/catalogue';
import { createHttpCatalogueClient } from './httpCatalogueClient';

export async function createCatalogueClient(
  config: CatalogueConfig,
  http: HttpClient,
): Promise<CatalogueClient> {
  switch (config.source) {
    case 'api':
      return createHttpCatalogueClient(http);
    case 'demo': {
      const { createDemoCatalogueClient } = await import('./demoCatalogueClient');
      return createDemoCatalogueClient(config.demoScenario);
    }
  }
}
