import type { HttpClient } from '../../../api/httpClient';
import type { CatalogueClient } from '../domain/catalogue';
import { parseCataloguePage } from './cataloguePayload';

export const publishedArticlesPath = '/v1/articles';

export function createHttpCatalogueClient(http: HttpClient): CatalogueClient {
  return {
    async listPublishedArticles({ page, pageSize }, signal) {
      const payload = await http.getJson(publishedArticlesPath, {
        query: { page: String(page), taille: String(pageSize) },
        signal,
      });
      return parseCataloguePage(payload);
    },
  };
}
