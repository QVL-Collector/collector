const preReservedArticleIds = ['demo-004'] as const;

const reservedArticleIds = new Set<string>(preReservedArticleIds);

export function isDemoArticleReserved(id: string): boolean {
  return reservedArticleIds.has(id);
}

export function markDemoArticleReserved(id: string): void {
  reservedArticleIds.add(id);
}
