import { useCallback } from 'react';
import { useSearchParams } from 'react-router';
import { preserveFocusState } from '../../../navigation/focusPreservation';
import { categoryParameter } from '../cataloguePaths';
import { isCategoryId } from '../domain/catalogue';

const pageParameter = 'page';

function parseCategoryId(raw: string | null): string | null {
  return raw !== null && isCategoryId(raw) ? raw : null;
}

function withCategory(current: URLSearchParams, categoryId: string | null): URLSearchParams {
  const params = new URLSearchParams(current);
  params.delete(pageParameter);
  if (categoryId === null) {
    params.delete(categoryParameter);
  } else {
    params.set(categoryParameter, categoryId);
  }
  return params;
}

export type CatalogueCategoryParam = {
  readonly categoryId: string | null;
  readonly selectCategory: (categoryId: string | null) => void;
  readonly clearCategory: () => void;
};

export function useCatalogueCategoryParam(): CatalogueCategoryParam {
  const [searchParams, setSearchParams] = useSearchParams();
  const categoryId = parseCategoryId(searchParams.get(categoryParameter));
  const selectCategory = useCallback(
    (next: string | null) => {
      setSearchParams((current) => withCategory(current, next), { state: preserveFocusState });
    },
    [setSearchParams],
  );
  const clearCategory = useCallback(() => {
    setSearchParams((current) => withCategory(current, null));
  }, [setSearchParams]);
  return { categoryId, selectCategory, clearCategory };
}
