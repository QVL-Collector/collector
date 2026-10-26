import type { HttpClient } from '../../../api/httpClient';
import type { CatalogueConfig } from '../../../config/runtimeConfig';
import type { SellerClient } from '../domain/sellerArticle';
import { createHttpSellerClient } from './httpSellerClient';

export async function createSellerClient(
  config: CatalogueConfig,
  http: HttpClient,
): Promise<SellerClient> {
  switch (config.source) {
    case 'api':
      return createHttpSellerClient(http);
    case 'demo': {
      const { createDemoSellerClient } = await import('./demoSellerClient');
      return createDemoSellerClient(config.demoScenario);
    }
  }
}
