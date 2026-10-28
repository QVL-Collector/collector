import { useCallback, useState } from 'react';
import { useDecimalSeparator } from '../../../i18n/useDecimalSeparator';
import type { SellerArticle } from '../domain/sellerArticle';
import {
  emptyArticleFormValues,
  haveSameValues,
  toArticleFormValues,
  type ArticleFormField,
  type ArticleFormValues,
} from './articleForm';

type FormState = {
  readonly values: ArticleFormValues;
  readonly initialValues: ArticleFormValues;
  readonly baseRevision: number | null;
};

export type ArticleFormState = {
  readonly values: ArticleFormValues;
  readonly baseRevision: number | null;
  readonly isDirty: boolean;
  readonly setValue: (field: ArticleFormField, value: string) => void;
  readonly reset: (article: SellerArticle) => void;
};

function stateFrom(article: SellerArticle | null, decimalSeparator: string): FormState {
  const values =
    article === null ? emptyArticleFormValues : toArticleFormValues(article, decimalSeparator);
  return { values, initialValues: values, baseRevision: article?.revision ?? null };
}

export function useArticleFormState(article: SellerArticle | null): ArticleFormState {
  const decimalSeparator = useDecimalSeparator();
  const [state, setState] = useState(() => stateFrom(article, decimalSeparator));
  const isDirty = !haveSameValues(state.values, state.initialValues);
  if (article !== null && article.revision !== state.baseRevision && !isDirty) {
    setState(stateFrom(article, decimalSeparator));
  }
  const setValue = useCallback((field: ArticleFormField, value: string) => {
    setState((current) => ({ ...current, values: { ...current.values, [field]: value } }));
  }, []);
  const reset = useCallback(
    (next: SellerArticle) => {
      setState(stateFrom(next, decimalSeparator));
    },
    [decimalSeparator],
  );
  return { values: state.values, baseRevision: state.baseRevision, isDirty, setValue, reset };
}
