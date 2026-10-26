import type { ControlReason } from '../../../shared/controlReason';
import type { Money } from '../../../shared/Money';

export const articleStates = [
  'brouillon',
  'en_controle',
  'publie',
  'refuse',
  'a_verifier',
] as const;

export type ArticleState = (typeof articleStates)[number];

export type SellerCategory = {
  readonly id: string;
  readonly label: string;
};

export type SellerArticle = {
  readonly id: string;
  readonly state: ArticleState;
  readonly revision: number;
  readonly title: string;
  readonly description: string;
  readonly category: SellerCategory | null;
  readonly price: Money | null;
  readonly shippingCost: Money | null;
  readonly reason: ControlReason | null;
  readonly updatedAt: Date;
};

export type SellerArticleQuery = {
  readonly page: number;
  readonly pageSize: number;
};

export type SellerArticlePage = {
  readonly articles: readonly SellerArticle[];
  readonly page: number;
  readonly pageSize: number;
  readonly totalItems: number;
};

export type ArticleDraft = {
  readonly title: string;
  readonly description: string;
  readonly categoryId: string | null;
  readonly priceInCents: number | null;
  readonly shippingCostInCents: number | null;
};

export type SellerClient = {
  readonly listMyArticles: (
    query: SellerArticleQuery,
    signal?: AbortSignal,
  ) => Promise<SellerArticlePage>;
  readonly listCategories: (signal?: AbortSignal) => Promise<readonly SellerCategory[]>;
  readonly getMyArticle: (id: string, signal?: AbortSignal) => Promise<SellerArticle>;
  readonly createDraft: (draft: ArticleDraft) => Promise<SellerArticle>;
  readonly updateDraft: (
    id: string,
    revision: number,
    draft: ArticleDraft,
  ) => Promise<SellerArticle>;
  readonly submitForControl: (id: string, revision: number) => Promise<SellerArticle>;
};

export const articleLimits = {
  titleMaxLength: 200,
  descriptionMaxLength: 5000,
  maxPriceInCents: 100_000_000,
  maxShippingCostInCents: 100_000,
} as const;

export const sellerArticlesPageSize = 20;

export const articleCurrency = 'EUR';

export function acceptsChanges(state: ArticleState): boolean {
  return state === 'brouillon' || state === 'refuse';
}

export function isUnderControl(state: ArticleState): boolean {
  return state === 'en_controle';
}
