import { createContext, type ReactNode } from 'react';
import type { PhotosClient } from '../domain/articlePhoto';

export const PhotosClientContext = createContext<PhotosClient | null>(null);

type PhotosClientProviderProps = {
  readonly client: PhotosClient | null;
  readonly children: ReactNode;
};

export function PhotosClientProvider({ client, children }: PhotosClientProviderProps) {
  return <PhotosClientContext value={client}>{children}</PhotosClientContext>;
}
