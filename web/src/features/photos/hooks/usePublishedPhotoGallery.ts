import { usePhotoGalleryView, type PhotoGalleryView } from './usePhotoGalleryView';
import { usePublishedArticlePhotos } from './usePublishedArticlePhotos';

export function usePublishedPhotoGallery(
  articleId: string,
  articleTitle: string,
): PhotoGalleryView {
  return usePhotoGalleryView(usePublishedArticlePhotos(articleId), articleTitle);
}
