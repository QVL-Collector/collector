export const cataloguePath = '/';

export const categoryParameter = 'categorie';

export const publishedArticleRoutePattern = '/articles/:articleId';

export function publishedArticlePath(id: string): string {
  return `/articles/${encodeURIComponent(id)}`;
}

export function categoryCataloguePath(categoryId: string): string {
  return `${cataloguePath}?${new URLSearchParams({ [categoryParameter]: categoryId }).toString()}`;
}
