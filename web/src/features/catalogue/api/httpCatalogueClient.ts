import type { HttpClient, QueryParameters } from '../../../api/httpClient';
import { parseCategories } from '../../../shared/marketplacePayload';
import type { CatalogueClient, CatalogueQuery } from '../domain/catalogue';
import { parseCataloguePage, parsePublishedArticle } from './cataloguePayload';

const publishedArticlesPath = '/v1/articles';
const categoriesPath = '/v1/categories';

function toQueryParameters({ page, pageSize, categoryId }: CatalogueQuery): QueryParameters {
  const pagination = { page: String(page), taille: String(pageSize) };
  return categoryId === null ? pagination : { categorie: categoryId, ...pagination };
}

function publishedArticlePath(id: string): string {
  return `${publishedArticlesPath}/${encodeURIComponent(id)}`;
}

export function createHttpCatalogueClient(http: HttpClient): CatalogueClient {
  return {
    async listPublishedArticles(query, signal) {
      const payload = await http.getJson(publishedArticlesPath, {
        query: toQueryParameters(query),
        signal,
      });
      return parseCataloguePage(payload);
    },
    async listCategories(signal) {
      return parseCategories(await http.getJson(categoriesPath, { signal }));
    },
    async getPublishedArticle(id, signal) {
      return parsePublishedArticle(await http.getJson(publishedArticlePath(id), { signal }));
    },
  };
}
