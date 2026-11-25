import { usePhotoGalleryView, type PhotoGalleryView } from './usePhotoGalleryView';
import { useReviewArticlePhotos } from './useReviewArticlePhotos';

export function useReviewPhotoGallery(articleId: string, articleTitle: string): PhotoGalleryView {
  return usePhotoGalleryView(useReviewArticlePhotos(articleId), articleTitle);
}
