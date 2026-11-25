import { skipToken, useQuery, type QueryKey, type UseQueryResult } from '@tanstack/react-query';
import { useCallback, useRef, useState } from 'react';
import type { ArticlePhoto } from '../domain/articlePhoto';
import { nextPresignedRefreshMs } from './presignedRefresh';

export type PhotoDisplayItem = {
  readonly photo: ArticlePhoto;
  readonly isBroken: boolean;
  readonly onError: () => void;
  readonly onLoad: () => void;
};

export type PresignedPhotosStatus = 'disabled' | 'loading' | 'error' | 'ready';

export type PresignedPhotosModel = {
  readonly status: PresignedPhotosStatus;
  readonly items: readonly PhotoDisplayItem[];
  readonly count: number;
  readonly retry: () => void;
};

export type FetchPhotos = (signal: AbortSignal) => Promise<readonly ArticlePhoto[]>;

function statusOf(
  query: UseQueryResult<readonly ArticlePhoto[]>,
  isEnabled: boolean,
): PresignedPhotosStatus {
  if (!isEnabled) {
    return 'disabled';
  }
  if (query.data !== undefined) {
    return 'ready';
  }
  return query.isError ? 'error' : 'loading';
}

export function usePresignedPhotos(
  queryKey: QueryKey,
  fetchPhotos: FetchPhotos | null,
): PresignedPhotosModel {
  const query = useQuery({
    queryKey,
    queryFn: fetchPhotos === null ? skipToken : ({ signal }) => fetchPhotos(signal),
    refetchInterval: (current) => nextPresignedRefreshMs(current.state.data, Date.now()),
  });
  const { refetch } = query;
  const refreshedPhotoIds = useRef(new Set<string>());
  const [brokenUrls, setBrokenUrls] = useState<ReadonlySet<string>>(() => new Set());

  const markBroken = useCallback((url: string) => {
    setBrokenUrls((current) => new Set(current).add(url));
  }, []);

  const refreshAfterFailure = useCallback(
    async (photo: ArticlePhoto) => {
      if (refreshedPhotoIds.current.has(photo.id)) {
        markBroken(photo.url);
        return;
      }
      refreshedPhotoIds.current.add(photo.id);
      const { data } = await refetch();
      const refreshed = data?.find(({ id }) => id === photo.id);
      if (refreshed === undefined || refreshed.url === photo.url) {
        markBroken(photo.url);
      }
    },
    [markBroken, refetch],
  );

  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);

  const photos = query.data ?? [];
  const items = photos.map(
    (photo): PhotoDisplayItem => ({
      photo,
      isBroken: brokenUrls.has(photo.url),
      onError: () => {
        void refreshAfterFailure(photo);
      },
      onLoad: () => {
        refreshedPhotoIds.current.delete(photo.id);
      },
    }),
  );

  return { status: statusOf(query, fetchPhotos !== null), items, count: photos.length, retry };
}
