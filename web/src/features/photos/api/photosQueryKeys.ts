export const photosQueryKeys = {
  sellerArticle: (articleId: string) => ['photos', 'seller', articleId] as const,
  publishedArticle: (articleId: string) => ['photos', 'published', articleId] as const,
  reviewArticle: (articleId: string) => ['photos', 'review', articleId] as const,
};
