export const sellerSpacePath = '/vendeur';

export const newArticlePath = `${sellerSpacePath}/articles/nouveau`;

export const articleRoutePattern = `${sellerSpacePath}/articles/:articleId`;

export function articlePath(id: string): string {
  return `${sellerSpacePath}/articles/${encodeURIComponent(id)}`;
}
