import { createContext, type ReactNode } from 'react';
import type { CatalogueClient } from '../domain/catalogue';

export const CatalogueClientContext = createContext<CatalogueClient | null>(null);

type CatalogueClientProviderProps = {
  readonly client: CatalogueClient;
  readonly children: ReactNode;
};

export function CatalogueClientProvider({ client, children }: CatalogueClientProviderProps) {
  return <CatalogueClientContext value={client}>{children}</CatalogueClientContext>;
}
