import { useCallback } from 'react';
import { useSearchParams } from 'react-router';

const pageParameter = 'page';
const pageNumberPattern = /^[1-9]\d{0,5}$/;

function parsePageNumber(raw: string | null): number {
  return raw !== null && pageNumberPattern.test(raw) ? Number(raw) : 1;
}

export type CataloguePageParam = {
  readonly page: number;
  readonly setPage: (page: number) => void;
};

export function useCataloguePageParam(): CataloguePageParam {
  const [searchParams, setSearchParams] = useSearchParams();
  const page = parsePageNumber(searchParams.get(pageParameter));
  const setPage = useCallback(
    (next: number) => {
      setSearchParams((current) => {
        const params = new URLSearchParams(current);
        if (next <= 1) {
          params.delete(pageParameter);
        } else {
          params.set(pageParameter, String(next));
        }
        return params;
      });
    },
    [setSearchParams],
  );
  return { page, setPage };
}
