import { createHttpClient } from '../api/httpClient';
import { anonymousSession, type AuthSession } from '../auth/AuthSession';
import { defaultInteractionsApiUrl, type RuntimeConfig } from '../config/runtimeConfig';
import { createAdminClient } from '../features/admin/api/createAdminClient';
import type { AdminClient } from '../features/admin/domain/reviewArticle';
import { createCatalogueClient } from '../features/catalogue/api/createCatalogueClient';
import type { CatalogueClient } from '../features/catalogue/domain/catalogue';
import { createNotificationsClient } from '../features/notifications/api/createNotificationsClient';
import type { NotificationsClient } from '../features/notifications/domain/sellerNotification';
import { createSellerClient } from '../features/seller/api/createSellerClient';
import type { SellerClient } from '../features/seller/domain/sellerArticle';

export type AppServices = {
  readonly catalogueClient: CatalogueClient;
  readonly sellerClient: SellerClient;
  readonly adminClient?: AdminClient;
  readonly notificationsClient?: NotificationsClient;
  readonly authSession?: AuthSession;
};

export type AppServicesConfig = Pick<RuntimeConfig, 'marketplaceApiUrl' | 'catalogue'> &
  Partial<Pick<RuntimeConfig, 'interactionsApiUrl'>>;

export async function createAppServices(
  config: AppServicesConfig,
  session: AuthSession = anonymousSession,
): Promise<AppServices> {
  const marketplaceHttp = createHttpClient({
    baseUrl: config.marketplaceApiUrl,
    getAccessToken: session.getAccessToken,
  });
  const interactionsHttp = createHttpClient({
    baseUrl: config.interactionsApiUrl ?? defaultInteractionsApiUrl,
    getAccessToken: session.getAccessToken,
  });
  const [catalogueClient, sellerClient, adminClient, notificationsClient] = await Promise.all([
    createCatalogueClient(config.catalogue, marketplaceHttp),
    createSellerClient(config.catalogue, marketplaceHttp),
    createAdminClient(config.catalogue, marketplaceHttp),
    createNotificationsClient(config.catalogue, interactionsHttp),
  ]);
  return { catalogueClient, sellerClient, adminClient, notificationsClient, authSession: session };
}
