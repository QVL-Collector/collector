export const sellerQueryKeys = {
  categories: () => ['seller', 'categories'] as const,
  articlePages: () => ['seller', 'article-pages'] as const,
  articlePage: (page: number, pageSize: number) =>
    ['seller', 'article-pages', page, pageSize] as const,
  article: (id: string) => ['seller', 'article', id] as const,
};
