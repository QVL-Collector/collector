import { QueryClientProvider, type QueryClient } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { anonymousSession } from '../auth/AuthSession';
import { AuthSessionProvider } from '../auth/AuthSessionContext';
import { AdminClientProvider } from '../features/admin/api/AdminClientContext';
import { CatalogueClientProvider } from '../features/catalogue/api/CatalogueClientContext';
import { SellerClientProvider } from '../features/seller/api/SellerClientContext';
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
      <AuthSessionProvider session={services.authSession ?? anonymousSession}>
        <QueryClientProvider client={queryClient}>
          <CatalogueClientProvider client={services.catalogueClient}>
            <SellerClientProvider client={services.sellerClient}>
              <AdminClientProvider client={services.adminClient ?? null}>
                {children}
              </AdminClientProvider>
            </SellerClientProvider>
          </CatalogueClientProvider>
        </QueryClientProvider>
      </AuthSessionProvider>
    </AppThemeRoot>
  );
}
