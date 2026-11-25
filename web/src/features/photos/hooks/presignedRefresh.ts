import type { ArticlePhoto } from '../domain/articlePhoto';

const refreshMarginMs = 60_000;
const minimumDelayMs = 5_000;

export function nextPresignedRefreshMs(
  photos: readonly ArticlePhoto[] | undefined,
  nowMs: number,
): number | false {
  if (photos === undefined || photos.length === 0) {
    return false;
  }
  const earliestExpiryMs = Math.min(...photos.map(({ expiresAt }) => expiresAt.getTime()));
  return Math.max(earliestExpiryMs - refreshMarginMs - nowMs, minimumDelayMs);
}
