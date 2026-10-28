import type { HttpClient } from '../../../api/httpClient';
import type { SellerClient } from '../domain/sellerArticle';
import { toDraftPayload, toRevisionPayload } from './draftPayload';
import {
  parseSellerArticle,
  parseSellerArticlePage,
  parseSellerCategories,
} from './sellerArticlePayload';

const categoriesPath = '/v1/categories';
const myArticlesPath = '/v1/vendeur/articles';

function myArticlePath(id: string): string {
  return `${myArticlesPath}/${encodeURIComponent(id)}`;
}

export function createHttpSellerClient(http: HttpClient): SellerClient {
  return {
    async listMyArticles({ page, pageSize }, signal) {
      const payload = await http.getJson(myArticlesPath, {
        query: { page: String(page), taille: String(pageSize) },
        signal,
      });
      return parseSellerArticlePage(payload);
    },
    async listCategories(signal) {
      return parseSellerCategories(await http.getJson(categoriesPath, { signal }));
    },
    async getMyArticle(id, signal) {
      return parseSellerArticle(await http.getJson(myArticlePath(id), { signal }));
    },
    async createDraft(draft) {
      return parseSellerArticle(await http.sendJson('POST', myArticlesPath, toDraftPayload(draft)));
    },
    async updateDraft(id, revision, draft) {
      const body = { ...toRevisionPayload(revision), ...toDraftPayload(draft) };
      return parseSellerArticle(await http.sendJson('PUT', myArticlePath(id), body));
    },
    async submitForControl(id, revision) {
      const path = `${myArticlePath(id)}/soumission`;
      return parseSellerArticle(await http.sendJson('POST', path, toRevisionPayload(revision)));
    },
  };
}
