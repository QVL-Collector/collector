import { useCallback, useMemo, useRef } from 'react';
import type { ArticleFormField } from './articleForm';

export type Focusable = {
  readonly focus: () => void;
};

export type FieldRefCallback = (element: Focusable | null) => void;

export type ArticleFieldFocus = {
  readonly refs: Readonly<Record<ArticleFormField, FieldRefCallback>>;
  readonly focusField: (field: ArticleFormField) => void;
};

export function useArticleFieldFocus(): ArticleFieldFocus {
  const elements = useRef(new Map<ArticleFormField, Focusable>());
  const refs = useMemo(() => {
    const callbackFor =
      (field: ArticleFormField): FieldRefCallback =>
      (element) => {
        if (element === null) {
          elements.current.delete(field);
        } else {
          elements.current.set(field, element);
        }
      };
    return {
      title: callbackFor('title'),
      description: callbackFor('description'),
      categoryId: callbackFor('categoryId'),
      price: callbackFor('price'),
      shippingCost: callbackFor('shippingCost'),
    };
  }, []);
  const focusField = useCallback((field: ArticleFormField) => {
    elements.current.get(field)?.focus();
  }, []);
  return { refs, focusField };
}
