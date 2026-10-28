import type { RefusalReason } from '../../../shared/articleReason';
import type { ControlReason } from '../../../shared/controlReason';
import type { Money } from '../../../shared/Money';

export type ReviewCategory = {
  readonly id: string;
  readonly label: string;
};

export type ReviewArticle = {
  readonly id: string;
  readonly revision: number;
  readonly title: string;
  readonly description: string;
  readonly category: ReviewCategory | null;
  readonly price: Money | null;
  readonly shippingCost: Money | null;
  readonly controlReason: ControlReason;
  readonly sellerId: string;
  readonly updatedAt: Date;
};

export type ReviewQuery = {
  readonly page: number;
  readonly pageSize: number;
};

export type ReviewArticlePage = {
  readonly articles: readonly ReviewArticle[];
  readonly page: number;
  readonly pageSize: number;
  readonly totalItems: number;
};

export type ReviewDecision =
  | { readonly kind: 'publish' }
  | { readonly kind: 'refuse'; readonly reason: RefusalReason };

export const decidedStates = ['publie', 'refuse'] as const;

export type DecidedState = (typeof decidedStates)[number];

export type DecisionOutcome = {
  readonly id: string;
  readonly revision: number;
  readonly state: DecidedState;
};

export type AdminClient = {
  readonly listArticlesToReview: (
    query: ReviewQuery,
    signal?: AbortSignal,
  ) => Promise<ReviewArticlePage>;
  readonly decide: (
    id: string,
    revision: number,
    decision: ReviewDecision,
  ) => Promise<DecisionOutcome>;
};

export const reviewPageSize = 20;
