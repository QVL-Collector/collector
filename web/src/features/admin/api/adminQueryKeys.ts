export const adminQueryKeys = {
  reviewPages: () => ['admin', 'review-pages'] as const,
  reviewPage: (page: number, pageSize: number) =>
    ['admin', 'review-pages', page, pageSize] as const,
};
