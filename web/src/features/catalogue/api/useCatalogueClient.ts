import { use } from 'react';
import type { CatalogueClient } from '../domain/catalogue';
import { CatalogueClientContext } from './CatalogueClientContext';

export function useCatalogueClient(): CatalogueClient {
  const client = use(CatalogueClientContext);
  if (client === null) {
    throw new Error('CatalogueClientProvider is missing');
  }
  return client;
}
