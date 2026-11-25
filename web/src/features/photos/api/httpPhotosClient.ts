import type { HttpClient } from '../../../api/httpClient';
import type { PhotoLimits, PhotosClient } from '../domain/articlePhoto';
import { parsePhoto, parsePhotoList } from './photoPayload';

const sellerArticlesPath = '/v1/vendeur/articles';
const publishedArticlesPath = '/v1/articles';
const reviewArticlesPath = '/v1/admin/articles';

function photosPath(articlesPath: string, articleId: string): string {
  return `${articlesPath}/${encodeURIComponent(articleId)}/photos`;
}

export function createHttpPhotosClient(http: HttpClient, limits: PhotoLimits): PhotosClient {
  const listPhotos = async (articlesPath: string, articleId: string, signal?: AbortSignal) =>
    parsePhotoList(await http.getJson(photosPath(articlesPath, articleId), { signal }));

  return {
    limits,
    listMyArticlePhotos: (articleId, signal) => listPhotos(sellerArticlesPath, articleId, signal),
    async uploadPhoto(articleId, { file, mediaType, onProgress, signal }) {
      const path = photosPath(sellerArticlesPath, articleId);
      return parsePhoto(await http.sendBinary(path, file, mediaType, { onProgress, signal }));
    },
    async deletePhoto(articleId, photoId) {
      const path = `${photosPath(sellerArticlesPath, articleId)}/${encodeURIComponent(photoId)}`;
      await http.deleteResource(path);
    },
    listPublishedArticlePhotos: (articleId, signal) =>
      listPhotos(publishedArticlesPath, articleId, signal),
    listReviewArticlePhotos: (articleId, signal) =>
      listPhotos(reviewArticlesPath, articleId, signal),
  };
}
