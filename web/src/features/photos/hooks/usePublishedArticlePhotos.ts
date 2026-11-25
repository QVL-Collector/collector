import { photosQueryKeys } from '../api/photosQueryKeys';
import { usePhotosClient } from '../api/usePhotosClient';
import { usePresignedPhotos, type PresignedPhotosModel } from './usePresignedPhotos';

export function usePublishedArticlePhotos(articleId: string): PresignedPhotosModel {
  const client = usePhotosClient();
  return usePresignedPhotos(
    photosQueryKeys.publishedArticle(articleId),
    client === null ? null : (signal) => client.listPublishedArticlePhotos(articleId, signal),
  );
}
