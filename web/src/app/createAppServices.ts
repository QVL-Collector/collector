import { createHttpClient } from '../api/httpClient';
import { anonymousSession, type AuthSession } from '../auth/AuthSession';
import type { RuntimeConfig } from '../config/runtimeConfig';
import { createCatalogueClient } from '../features/catalogue/api/createCatalogueClient';
import type { CatalogueClient } from '../features/catalogue/domain/catalogue';
import { createSellerClient } from '../features/seller/api/createSellerClient';
import type { SellerClient } from '../features/seller/domain/sellerArticle';

export type AppServices = {
  readonly catalogueClient: CatalogueClient;
  readonly sellerClient?: SellerClient;
  readonly authSession?: AuthSession;
};

export type AppServicesConfig = Pick<RuntimeConfig, 'marketplaceApiUrl' | 'catalogue'>;

export async function createAppServices(
  config: AppServicesConfig,
  session: AuthSession = anonymousSession,
): Promise<AppServices> {
  const marketplaceHttp = createHttpClient({
    baseUrl: config.marketplaceApiUrl,
    getAccessToken: session.getAccessToken,
  });
  const [catalogueClient, sellerClient] = await Promise.all([
    createCatalogueClient(config.catalogue, marketplaceHttp),
    createSellerClient(config.catalogue, marketplaceHttp),
  ]);
  return { catalogueClient, sellerClient, authSession: session };
}
