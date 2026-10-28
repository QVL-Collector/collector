import { useMemo } from 'react';
import type { SellerCategory } from '../domain/sellerArticle';
import { useSellerCategories } from './useSellerCategories';

export type CategoryOptions = {
  readonly options: readonly SellerCategory[];
  readonly unavailable: boolean;
};

export function useCategoryOptions(current: SellerCategory | null): CategoryOptions {
  const { data, isError } = useSellerCategories();
  const options = useMemo(() => {
    const categories = data ?? [];
    const isMissing = current !== null && !categories.some(({ id }) => id === current.id);
    return isMissing ? [current, ...categories] : categories;
  }, [current, data]);
  return { options, unavailable: isError && data === undefined };
}
