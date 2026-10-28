import { useId, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import type { ArticleFieldErrors, ArticleFormField } from './articleForm';
import type { FieldRefCallback } from './useArticleFieldFocus';
import type { ArticleFormState } from './useArticleFormState';

export type ArticleFieldModel = {
  readonly id: string;
  readonly value: string;
  readonly error: string | null;
  readonly inputRef: FieldRefCallback;
  readonly onChange: (value: string) => void;
};

export type ArticleFieldModels = Readonly<Record<ArticleFormField, ArticleFieldModel>>;

type ArticleFieldModelsOptions = {
  readonly form: ArticleFormState;
  readonly errors: ArticleFieldErrors;
  readonly refs: Readonly<Record<ArticleFormField, FieldRefCallback>>;
  readonly onFieldEdited: (field: ArticleFormField) => void;
};

export function useArticleFieldModels({
  form,
  errors,
  refs,
  onFieldEdited,
}: ArticleFieldModelsOptions): ArticleFieldModels {
  const { t } = useTranslation();
  const idPrefix = useId();
  const { values, setValue } = form;
  return useMemo(() => {
    const modelOf = (field: ArticleFormField): ArticleFieldModel => {
      const code = errors[field];
      return {
        id: `${idPrefix}-${field}`,
        value: values[field],
        error: code === undefined ? null : t(`seller.form.errors.${code}`),
        inputRef: refs[field],
        onChange: (value) => {
          setValue(field, value);
          onFieldEdited(field);
        },
      };
    };
    return {
      title: modelOf('title'),
      description: modelOf('description'),
      categoryId: modelOf('categoryId'),
      price: modelOf('price'),
      shippingCost: modelOf('shippingCost'),
    };
  }, [errors, idPrefix, onFieldEdited, refs, setValue, t, values]);
}
