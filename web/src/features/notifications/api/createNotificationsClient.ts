import type { HttpClient } from '../../../api/httpClient';
import type { CatalogueConfig } from '../../../config/runtimeConfig';
import type { NotificationsClient } from '../domain/sellerNotification';
import { createHttpNotificationsClient } from './httpNotificationsClient';

export async function createNotificationsClient(
  config: CatalogueConfig,
  http: HttpClient,
): Promise<NotificationsClient> {
  switch (config.source) {
    case 'api':
      return createHttpNotificationsClient(http);
    case 'demo': {
      const { createDemoNotificationsClient } = await import('./demoNotificationsClient');
      return createDemoNotificationsClient(config.demoScenario);
    }
  }
}
