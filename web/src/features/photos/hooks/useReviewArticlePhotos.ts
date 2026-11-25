import { photosQueryKeys } from '../api/photosQueryKeys';
import { usePhotosClient } from '../api/usePhotosClient';
import { usePresignedPhotos, type PresignedPhotosModel } from './usePresignedPhotos';

export function useReviewArticlePhotos(articleId: string): PresignedPhotosModel {
  const client = usePhotosClient();
  return usePresignedPhotos(
    photosQueryKeys.reviewArticle(articleId),
    client === null ? null : (signal) => client.listReviewArticlePhotos(articleId, signal),
  );
}
