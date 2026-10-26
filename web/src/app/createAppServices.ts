import { createHttpClient } from '../api/httpClient';
import { anonymousSession, type AuthSession } from '../auth/AuthSession';
import type { RuntimeConfig } from '../config/runtimeConfig';
import { createCatalogueClient } from '../features/catalogue/api/createCatalogueClient';
import type { CatalogueClient } from '../features/catalogue/domain/catalogue';

export type AppServices = {
  readonly catalogueClient: CatalogueClient;
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
  return {
    catalogueClient: await createCatalogueClient(config.catalogue, marketplaceHttp),
    authSession: session,
  };
}
