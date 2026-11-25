import { createHttpClient } from '../api/httpClient';
import { anonymousSession, type AuthSession } from '../auth/AuthSession';
import type { RuntimeConfig } from '../config/runtimeConfig';
import { createAdminClient } from '../features/admin/api/createAdminClient';
import type { AdminClient } from '../features/admin/domain/reviewArticle';
import { createCatalogueClient } from '../features/catalogue/api/createCatalogueClient';
import type { CatalogueClient } from '../features/catalogue/domain/catalogue';
import { createNotificationsClient } from '../features/notifications/api/createNotificationsClient';
import type { NotificationsClient } from '../features/notifications/domain/sellerNotification';
import { createOrdersClient } from '../features/orders/api/createOrdersClient';
import type { OrdersClient } from '../features/orders/domain/order';
import { createPhotosClient } from '../features/photos/api/createPhotosClient';
import { defaultPhotoLimits, type PhotosClient } from '../features/photos/domain/articlePhoto';
import { createSellerClient } from '../features/seller/api/createSellerClient';
import type { SellerClient } from '../features/seller/domain/sellerArticle';

export type AppServices = {
  readonly catalogueClient: CatalogueClient;
  readonly sellerClient: SellerClient;
  readonly adminClient?: AdminClient;
  readonly notificationsClient?: NotificationsClient;
  readonly ordersClient?: OrdersClient;
  readonly photosClient?: PhotosClient;
  readonly authSession?: AuthSession;
};

export type AppServicesConfig = Pick<
  RuntimeConfig,
  'marketplaceApiUrl' | 'interactionsApiUrl' | 'transactionsApiUrl' | 'catalogue' | 'photos'
>;

export async function createAppServices(
  config: AppServicesConfig,
  session: AuthSession = anonymousSession,
): Promise<AppServices> {
  const httpFor = (baseUrl: string) =>
    createHttpClient({ baseUrl, getAccessToken: session.getAccessToken });
  const marketplaceHttp = httpFor(config.marketplaceApiUrl);
  const [
    catalogueClient,
    sellerClient,
    adminClient,
    notificationsClient,
    ordersClient,
    photosClient,
  ] = await Promise.all([
    createCatalogueClient(config.catalogue, marketplaceHttp),
    createSellerClient(config.catalogue, marketplaceHttp),
    createAdminClient(config.catalogue, marketplaceHttp),
    createNotificationsClient(config.catalogue, httpFor(config.interactionsApiUrl)),
    createOrdersClient(config.catalogue, httpFor(config.transactionsApiUrl)),
    createPhotosClient(config.catalogue, marketplaceHttp, config.photos ?? defaultPhotoLimits),
  ]);
  return {
    catalogueClient,
    sellerClient,
    adminClient,
    notificationsClient,
    ordersClient,
    photosClient,
    authSession: session,
  };
}
