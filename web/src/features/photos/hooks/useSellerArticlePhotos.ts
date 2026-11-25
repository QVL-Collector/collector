import { photosQueryKeys } from '../api/photosQueryKeys';
import { usePhotosClient } from '../api/usePhotosClient';
import { usePresignedPhotos, type PresignedPhotosModel } from './usePresignedPhotos';

export function useSellerArticlePhotos(articleId: string): PresignedPhotosModel {
  const client = usePhotosClient();
  return usePresignedPhotos(
    photosQueryKeys.sellerArticle(articleId),
    client === null ? null : (signal) => client.listMyArticlePhotos(articleId, signal),
  );
}
