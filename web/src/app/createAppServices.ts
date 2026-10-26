import { createHttpClient } from '../api/httpClient';
import { anonymousSession, type AuthSession } from '../auth/AuthSession';
import type { RuntimeConfig } from '../config/runtimeConfig';
import { createCatalogueClient } from '../features/catalogue/api/createCatalogueClient';
import type { CatalogueClient } from '../features/catalogue/domain/catalogue';

export type AppServices = {
  readonly catalogueClient: CatalogueClient;
};

export async function createAppServices(
  config: RuntimeConfig,
  session: AuthSession = anonymousSession,
): Promise<AppServices> {
  const marketplaceHttp = createHttpClient({
    baseUrl: config.marketplaceApiUrl,
    getAccessToken: session.getAccessToken,
  });
  return {
    catalogueClient: await createCatalogueClient(config.catalogue, marketplaceHttp),
  };
}
