import { use } from 'react';
import type { PhotosClient } from '../domain/articlePhoto';
import { PhotosClientContext } from './PhotosClientContext';

export function usePhotosClient(): PhotosClient | null {
  return use(PhotosClientContext);
}
