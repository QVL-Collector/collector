import { useCallback } from 'react';
import type { ArticleCategory } from '../domain/catalogue';
import { useCatalogueCategories } from './useCatalogueCategories';

export const allCategoriesValue = '';

export type CategoryFilterModel = {
  readonly value: string;
  readonly options: readonly ArticleCategory[];
  readonly unavailable: boolean;
  readonly selectedLabel: string | null;
  readonly onChange: (value: string) => void;
};

function withSelected(
  categories: readonly ArticleCategory[],
  selectedId: string | null,
): readonly ArticleCategory[] {
  if (selectedId === null || categories.some((category) => category.id === selectedId)) {
    return categories;
  }
  return [...categories, { id: selectedId, label: selectedId }];
}

export function useCategoryFilter(
  selectedId: string | null,
  onSelect: (categoryId: string | null) => void,
): CategoryFilterModel {
  const { data: categories = [], isError } = useCatalogueCategories();
  const onChange = useCallback(
    (value: string) => {
      onSelect(value === allCategoriesValue ? null : value);
    },
    [onSelect],
  );
  return {
    value: selectedId ?? allCategoriesValue,
    options: withSelected(categories, selectedId),
    unavailable: isError,
    selectedLabel: categories.find((category) => category.id === selectedId)?.label ?? null,
    onChange,
  };
}
