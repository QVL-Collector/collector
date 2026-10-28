import { use } from 'react';
import type { AdminClient } from '../domain/reviewArticle';
import { AdminClientContext } from './AdminClientContext';

export function useAdminClient(): AdminClient {
  const client = use(AdminClientContext);
  if (client === null) {
    throw new Error('AdminClientProvider is missing');
  }
  return client;
}
