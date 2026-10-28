import type { HttpClient } from '../../../api/httpClient';
import type { AdminClient } from '../domain/reviewArticle';
import {
  parseDecisionOutcome,
  parseReviewArticlePage,
  toDecisionPayload,
} from './reviewArticlePayload';

const reviewArticlesPath = '/v1/admin/articles';

function decisionPath(id: string): string {
  return `${reviewArticlesPath}/${encodeURIComponent(id)}/decision`;
}

export function createHttpAdminClient(http: HttpClient): AdminClient {
  return {
    async listArticlesToReview({ page, pageSize }, signal) {
      const payload = await http.getJson(reviewArticlesPath, {
        query: { page: String(page), taille: String(pageSize) },
        signal,
      });
      return parseReviewArticlePage(payload);
    },
    async decide(id, revision, decision) {
      const body = toDecisionPayload(revision, decision);
      return parseDecisionOutcome(await http.sendJson('POST', decisionPath(id), body));
    },
  };
}
