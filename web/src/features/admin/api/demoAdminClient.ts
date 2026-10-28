import { ApiError } from '../../../api/ApiError';
import { waitForDemoLatency } from '../../../api/demoLatency';
import type { DemoScenario } from '../../../config/runtimeConfig';
import { adminHttpStatuses, adminProblemTypes } from '../domain/adminProblems';
import type { AdminClient, DecidedState, ReviewDecision } from '../domain/reviewArticle';
import {
  demoReviewArticleChangedBySellerId,
  demoReviewArticles,
  type DemoReviewArticle,
} from './demoReviewArticles';
import { parseDecisionOutcome, parseReviewArticlePage } from './reviewArticlePayload';

function problemError(status: number, type: string): ApiError {
  return new ApiError(status, { type, fieldProblems: [] });
}

function byOldestUpdate(left: DemoReviewArticle, right: DemoReviewArticle): number {
  return left.modifieLe.localeCompare(right.modifieLe);
}

function decidedStateOf(decision: ReviewDecision): DecidedState {
  return decision.kind === 'publish' ? 'publie' : 'refuse';
}

function initialArticles(scenario: DemoScenario): Map<string, DemoReviewArticle> {
  const articles = scenario === 'empty' ? [] : demoReviewArticles;
  return new Map(articles.map((article): [string, DemoReviewArticle] => [article.id, article]));
}

export function createDemoAdminClient(scenario: DemoScenario): AdminClient {
  const articles = initialArticles(scenario);
  const changedBySeller = new Set([demoReviewArticleChangedBySellerId]);

  const find = (id: string): DemoReviewArticle => {
    const article = articles.get(id);
    if (article === undefined) {
      throw problemError(adminHttpStatuses.conflict, adminProblemTypes.notUnderReview);
    }
    return article;
  };

  const simulateChangeBySeller = (article: DemoReviewArticle): void => {
    if (changedBySeller.delete(article.id)) {
      articles.set(article.id, {
        ...article,
        revision: article.revision + 1,
        modifieLe: new Date().toISOString(),
      });
      throw problemError(adminHttpStatuses.conflict, adminProblemTypes.staleRevision);
    }
  };

  return {
    async listArticlesToReview({ page, pageSize }, signal) {
      await waitForDemoLatency(signal);
      if (scenario === 'error') {
        throw new ApiError(adminHttpStatuses.serviceUnavailable);
      }
      const sorted = [...articles.values()].sort(byOldestUpdate);
      const start = (page - 1) * pageSize;
      return parseReviewArticlePage({
        elements: sorted.slice(start, start + pageSize),
        page,
        taille: pageSize,
        total: sorted.length,
      });
    },
    async decide(id, revision, decision) {
      await waitForDemoLatency();
      const article = find(id);
      simulateChangeBySeller(article);
      if (article.revision !== revision) {
        throw problemError(adminHttpStatuses.conflict, adminProblemTypes.staleRevision);
      }
      articles.delete(id);
      return parseDecisionOutcome({
        id: article.id,
        revision: article.revision,
        etat: decidedStateOf(decision),
      });
    },
  };
}
