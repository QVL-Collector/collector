import type { Money } from '../../../shared/Money';

export type ArticleCategory = {
  readonly id: string;
  readonly label: string;
};

export type CatalogueArticle = {
  readonly id: string;
  readonly title: string;
  readonly price: Money;
  readonly category: ArticleCategory;
};

export type PublishedArticle = CatalogueArticle & {
  readonly description: string;
  readonly shippingCost: Money | null;
};

export type CatalogueQuery = {
  readonly page: number;
  readonly pageSize: number;
  readonly categoryId: string | null;
};

export type CataloguePage = {
  readonly articles: readonly CatalogueArticle[];
  readonly page: number;
  readonly pageSize: number;
  readonly totalItems: number;
};

export type CatalogueClient = {
  readonly listPublishedArticles: (
    query: CatalogueQuery,
    signal?: AbortSignal,
  ) => Promise<CataloguePage>;
  readonly listCategories: (signal?: AbortSignal) => Promise<readonly ArticleCategory[]>;
  readonly getPublishedArticle: (id: string, signal?: AbortSignal) => Promise<PublishedArticle>;
};

export const cataloguePageSize = 12;

export const categoryIdPattern = /^[a-z0-9]+(-[a-z0-9]+)*$/;

export const categoryIdMaxLength = 64;

export function isCategoryId(value: string): boolean {
  return value.length <= categoryIdMaxLength && categoryIdPattern.test(value);
}
