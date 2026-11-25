import { useReviewPhotoGallery } from '../hooks/useReviewPhotoGallery';
import { ArticlePhotoGallery } from './ArticlePhotoGallery';

type ReviewArticlePhotosProps = {
  readonly articleId: string;
  readonly articleTitle: string;
};

export function ReviewArticlePhotos({ articleId, articleTitle }: ReviewArticlePhotosProps) {
  const view = useReviewPhotoGallery(articleId, articleTitle);
  return <ArticlePhotoGallery view={view} headingComponent="h4" maxWidth="20rem" />;
}
