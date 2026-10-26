import { QueryClientProvider, type QueryClient } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { CatalogueClientProvider } from '../features/catalogue/api/CatalogueClientContext';
import { AppThemeRoot } from './AppThemeRoot';
import type { AppServices } from './createAppServices';

type AppProvidersProps = {
  readonly services: AppServices;
  readonly queryClient: QueryClient;
  readonly children: ReactNode;
};

export function AppProviders({ services, queryClient, children }: AppProvidersProps) {
  return (
    <AppThemeRoot>
      <QueryClientProvider client={queryClient}>
        <CatalogueClientProvider client={services.catalogueClient}>
          {children}
        </CatalogueClientProvider>
      </QueryClientProvider>
    </AppThemeRoot>
  );
}
