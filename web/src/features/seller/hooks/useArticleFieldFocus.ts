import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type { ArticleFormField } from './articleForm';

export type Focusable = {
  readonly focus: () => void;
};

export type FieldRefCallback = (element: Focusable | null) => void;

export type ArticleFieldFocus = {
  readonly refs: Readonly<Record<ArticleFormField, FieldRefCallback>>;
  readonly focusField: (field: ArticleFormField) => void;
};

type FocusRequest = {
  readonly field: ArticleFormField;
};

export function useArticleFieldFocus(): ArticleFieldFocus {
  const elements = useRef(new Map<ArticleFormField, Focusable>());
  const [request, setRequest] = useState<FocusRequest | null>(null);
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
  useEffect(() => {
    if (request !== null) {
      elements.current.get(request.field)?.focus();
    }
  }, [request]);
  const focusField = useCallback((field: ArticleFormField) => {
    setRequest({ field });
  }, []);
  return { refs, focusField };
}
