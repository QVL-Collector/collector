import type { HttpClient } from '../../../api/httpClient';
import type { CatalogueConfig } from '../../../config/runtimeConfig';
import type { OrdersClient } from '../domain/order';
import { createHttpOrdersClient } from './httpOrdersClient';

export async function createOrdersClient(
  config: CatalogueConfig,
  http: HttpClient,
): Promise<OrdersClient> {
  switch (config.source) {
    case 'api':
      return createHttpOrdersClient(http);
    case 'demo': {
      const { createDemoOrdersClient } = await import('./demoOrdersClient');
      return createDemoOrdersClient(config.demoScenario);
    }
  }
}
