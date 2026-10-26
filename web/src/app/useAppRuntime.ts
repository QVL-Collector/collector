import type { QueryClient } from '@tanstack/react-query';
import { useState } from 'react';
import { createBrowserRouter } from 'react-router';
import { createQueryClient } from './createQueryClient';
import { appRoutes } from './routes';

export type AppRuntime = {
  readonly queryClient: QueryClient;
  readonly router: ReturnType<typeof createBrowserRouter>;
};

function createAppRuntime(): AppRuntime {
  return { queryClient: createQueryClient(), router: createBrowserRouter(appRoutes) };
}

export function useAppRuntime(): AppRuntime {
  const [runtime] = useState(createAppRuntime);
  return runtime;
}
