import { RouterProvider } from 'react-router';
import { AppProviders } from './AppProviders';
import type { AppServices } from './createAppServices';
import { useAppRuntime } from './useAppRuntime';

type AppProps = {
  readonly services: AppServices;
};

export function App({ services }: AppProps) {
  const { queryClient, router } = useAppRuntime();
  return (
    <AppProviders services={services} queryClient={queryClient}>
      <RouterProvider router={router} />
    </AppProviders>
  );
}
