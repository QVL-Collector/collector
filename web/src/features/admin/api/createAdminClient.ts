import type { HttpClient } from '../../../api/httpClient';
import type { CatalogueConfig } from '../../../config/runtimeConfig';
import type { AdminClient } from '../domain/reviewArticle';
import { createHttpAdminClient } from './httpAdminClient';

export async function createAdminClient(
  config: CatalogueConfig,
  http: HttpClient,
): Promise<AdminClient> {
  switch (config.source) {
    case 'api':
      return createHttpAdminClient(http);
    case 'demo': {
      const { createDemoAdminClient } = await import('./demoAdminClient');
      return createDemoAdminClient(config.demoScenario);
    }
  }
}
