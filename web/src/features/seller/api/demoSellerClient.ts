import { ApiError } from '../../../api/ApiError';
import { waitForDemoLatency } from '../../../api/demoLatency';
import type { FieldProblem } from '../../../api/problemDetails';
import type { DemoScenario } from '../../../config/runtimeConfig';
import { acceptsChanges, type ArticleDraft, type SellerClient } from '../domain/sellerArticle';
import { httpStatuses, sellerProblemTypes } from '../domain/sellerProblems';
import { controlDemoArticle } from './demoControl';
import { validateDemoDraft } from './demoDraftValidation';
import {
  demoArticleModifiedElsewhereId,
  demoCategories,
  demoSellerArticles,
  type DemoArticle,
} from './demoSellerData';
import { toAmountPayload } from './draftPayload';
import {
  parseSellerArticle,
  parseSellerArticlePage,
  parseSellerCategories,
} from './sellerArticlePayload';

const controlDelayMs = 2500;

type DraftFields = Pick<DemoArticle, 'titre' | 'description' | 'categorie' | 'prix' | 'fraisDePort'>;

function problemError(
  status: number,
  type: string,
  fieldProblems: readonly FieldProblem[] = [],
): ApiError {
  return new ApiError(status, { type, fieldProblems });
}

function draftFields(draft: ArticleDraft): DraftFields {
  return {
    titre: draft.title,
    description: draft.description,
    categorie: demoCategories.find(({ id }) => id === draft.categoryId) ?? null,
    prix: toAmountPayload(draft.priceInCents),
    fraisDePort: toAmountPayload(draft.shippingCostInCents),
  };
}

function nowIso(): string {
  return new Date().toISOString();
}

function byMostRecentUpdate(left: DemoArticle, right: DemoArticle): number {
  return right.modifieLe.localeCompare(left.modifieLe);
}

function expectValidDraft(draft: ArticleDraft): void {
  const fieldProblems = validateDemoDraft(draft);
  if (fieldProblems.length > 0) {
    throw problemError(httpStatuses.badRequest, sellerProblemTypes.invalidInput, fieldProblems);
  }
}

function expectChangeable(article: DemoArticle, problemType: string): void {
  if (!acceptsChanges(article.etat)) {
    throw problemError(httpStatuses.conflict, problemType);
  }
}

function expectRevision(article: DemoArticle, revision: number): void {
  if (article.revision !== revision) {
    throw problemError(httpStatuses.conflict, sellerProblemTypes.staleRevision);
  }
}

function initialArticles(scenario: DemoScenario): Map<string, DemoArticle> {
  const articles = scenario === 'empty' ? [] : demoSellerArticles;
  return new Map(articles.map((article): [string, DemoArticle] => [article.id, article]));
}

export function createDemoSellerClient(scenario: DemoScenario): SellerClient {
  const articles = initialArticles(scenario);
  const modifiedElsewhere = new Set([demoArticleModifiedElsewhereId]);

  const save = (article: DemoArticle): DemoArticle => {
    articles.set(article.id, article);
    return article;
  };

  const find = (id: string): DemoArticle => {
    const article = articles.get(id);
    if (article === undefined) {
      throw new ApiError(httpStatuses.notFound);
    }
    return article;
  };

  const simulateChangeFromAnotherTab = (article: DemoArticle): void => {
    if (modifiedElsewhere.delete(article.id)) {
      save({ ...article, revision: article.revision + 1, modifieLe: nowIso() });
      throw problemError(httpStatuses.conflict, sellerProblemTypes.staleRevision);
    }
  };

  const scheduleControl = (id: string, revision: number): void => {
    window.setTimeout(() => {
      const article = articles.get(id);
      if (article === undefined || article.revision !== revision || article.etat !== 'en_controle') {
        return;
      }
      const outcome = controlDemoArticle(article);
      save({ ...article, etat: outcome.state, motif: outcome.reason, modifieLe: nowIso() });
    }, controlDelayMs);
  };

  return {
    async listMyArticles({ page, pageSize }, signal) {
      await waitForDemoLatency(signal);
      if (scenario === 'error') {
        throw new ApiError(httpStatuses.serviceUnavailable);
      }
      const sorted = [...articles.values()].sort(byMostRecentUpdate);
      const start = (page - 1) * pageSize;
      return parseSellerArticlePage({
        elements: sorted.slice(start, start + pageSize),
        page,
        taille: pageSize,
        total: sorted.length,
      });
    },
    async listCategories(signal) {
      await waitForDemoLatency(signal);
      return parseSellerCategories({ elements: demoCategories });
    },
    async getMyArticle(id, signal) {
      await waitForDemoLatency(signal);
      return parseSellerArticle(find(id));
    },
    async createDraft(draft) {
      await waitForDemoLatency();
      expectValidDraft(draft);
      const now = nowIso();
      return parseSellerArticle(
        save({
          id: crypto.randomUUID(),
          etat: 'brouillon',
          revision: 1,
          ...draftFields(draft),
          motif: null,
          creeLe: now,
          modifieLe: now,
        }),
      );
    },
    async updateDraft(id, revision, draft) {
      await waitForDemoLatency();
      const article = find(id);
      simulateChangeFromAnotherTab(article);
      expectChangeable(article, sellerProblemTypes.notModifiable);
      expectRevision(article, revision);
      expectValidDraft(draft);
      return parseSellerArticle(
        save({
          ...article,
          ...draftFields(draft),
          revision: article.revision + 1,
          modifieLe: nowIso(),
        }),
      );
    },
    async submitForControl(id, revision) {
      await waitForDemoLatency();
      const article = find(id);
      simulateChangeFromAnotherTab(article);
      expectChangeable(article, sellerProblemTypes.notSubmittable);
      expectRevision(article, revision);
      const submitted = save({
        ...article,
        etat: 'en_controle',
        motif: null,
        revision: article.revision + 1,
        modifieLe: nowIso(),
      });
      scheduleControl(id, submitted.revision);
      return parseSellerArticle(submitted);
    },
  };
}
