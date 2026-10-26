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

export type CatalogueQuery = {
  readonly page: number;
  readonly pageSize: number;
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
};

export const cataloguePageSize = 12;
