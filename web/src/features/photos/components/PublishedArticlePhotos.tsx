import { usePublishedPhotoGallery } from '../hooks/usePublishedPhotoGallery';
import { ArticlePhotoGallery } from './ArticlePhotoGallery';

type PublishedArticlePhotosProps = {
  readonly articleId: string;
  readonly articleTitle: string;
};

export function PublishedArticlePhotos({ articleId, articleTitle }: PublishedArticlePhotosProps) {
  const view = usePublishedPhotoGallery(articleId, articleTitle);
  return <ArticlePhotoGallery view={view} headingComponent="h2" maxWidth="40rem" />;
}
