import type { RefusalReason } from '../../../shared/articleReason';
import { controlReasons } from '../../../shared/controlReason';
import {
  parseCategory,
  parseDate,
  parseMoney,
  parseNullable,
  parseUuid,
} from '../../../shared/marketplacePayload';
import {
  expectArray,
  expectInteger,
  expectNonEmptyString,
  expectOneOf,
  expectRecord,
  expectString,
} from '../../../shared/payload';
import {
  decidedStates,
  type DecisionOutcome,
  type ReviewArticle,
  type ReviewArticlePage,
  type ReviewDecision,
} from '../domain/reviewArticle';

export type DecisionPayload =
  | { readonly revision: number; readonly decision: 'publier' }
  | { readonly revision: number; readonly decision: 'refuser'; readonly motif: RefusalReason };

export function toDecisionPayload(revision: number, decision: ReviewDecision): DecisionPayload {
  switch (decision.kind) {
    case 'publish':
      return { revision, decision: 'publier' };
    case 'refuse':
      return { revision, decision: 'refuser', motif: decision.reason };
  }
}

export function parseReviewArticle(value: unknown, path = 'article'): ReviewArticle {
  const article = expectRecord(value, path);
  return {
    id: parseUuid(article.id, `${path}.id`),
    revision: expectInteger(article.revision, `${path}.revision`, 1),
    title: expectString(article.titre, `${path}.titre`),
    description: expectString(article.description, `${path}.description`),
    category: parseNullable(article.categorie, (category) =>
      parseCategory(category, `${path}.categorie`),
    ),
    price: parseNullable(article.prix, (price) => parseMoney(price, `${path}.prix`)),
    shippingCost: parseNullable(article.fraisDePort, (shipping) =>
      parseMoney(shipping, `${path}.fraisDePort`),
    ),
    controlReason: expectOneOf(article.motif, controlReasons, `${path}.motif`),
    sellerId: expectNonEmptyString(article.vendeurId, `${path}.vendeurId`),
    updatedAt: parseDate(article.modifieLe, `${path}.modifieLe`),
  };
}

export function parseReviewArticlePage(payload: unknown): ReviewArticlePage {
  const root = expectRecord(payload, 'articles');
  return {
    articles: expectArray(root.elements, 'elements').map((item, index) =>
      parseReviewArticle(item, `elements[${String(index)}]`),
    ),
    page: expectInteger(root.page, 'page', 1),
    pageSize: expectInteger(root.taille, 'taille', 1),
    totalItems: expectInteger(root.total, 'total', 0),
  };
}

export function parseDecisionOutcome(payload: unknown): DecisionOutcome {
  const article = expectRecord(payload, 'article');
  return {
    id: parseUuid(article.id, 'article.id'),
    revision: expectInteger(article.revision, 'article.revision', 1),
    state: expectOneOf(article.etat, decidedStates, 'article.etat'),
  };
}
