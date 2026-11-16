const preReservedArticleIds = ['demo-004'] as const;

const reservedUntilByArticleId = new Map<string, number>(
  preReservedArticleIds.map((id): [string, number] => [id, Number.POSITIVE_INFINITY]),
);

export function isDemoArticleReserved(id: string, now: Date = new Date()): boolean {
  const reservedUntil = reservedUntilByArticleId.get(id);
  return reservedUntil !== undefined && now.getTime() < reservedUntil;
}

export function markDemoArticleReserved(id: string, until: Date): void {
  reservedUntilByArticleId.set(id, until.getTime());
}
