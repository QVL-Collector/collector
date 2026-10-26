import { QueryClient } from '@tanstack/react-query';
import { isApiError } from '../api/ApiError';

const maxRetries = 1;
const staleTimeMs = 30_000;
const clientErrorStatuses = { min: 400, max: 499 } as const;

function isClientError(error: unknown): boolean {
  return (
    isApiError(error) &&
    error.status !== null &&
    error.status >= clientErrorStatuses.min &&
    error.status <= clientErrorStatuses.max
  );
}

function shouldRetry(failureCount: number, error: unknown): boolean {
  return failureCount < maxRetries && !isClientError(error);
}

export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: shouldRetry,
        staleTime: staleTimeMs,
        refetchOnWindowFocus: false,
      },
    },
  });
}
