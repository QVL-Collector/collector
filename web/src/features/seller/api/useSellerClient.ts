import { use } from 'react';
import type { SellerClient } from '../domain/sellerArticle';
import { SellerClientContext } from './SellerClientContext';

export function useSellerClient(): SellerClient {
  const client = use(SellerClientContext);
  if (client === null) {
    throw new Error('SellerClientProvider is missing');
  }
  return client;
}
