import { useCallback, useState, type FormEvent } from 'react';
import { useTranslation } from 'react-i18next';
import type { SellerArticle } from '../domain/sellerArticle';
import { toArticleFailure } from './articleFailure';
import {
  countFieldErrors,
  firstInvalidField,
  toArticleDraft,
  withoutFieldError,
  type ArticleFieldErrors,
  type ArticleFormField,
} from './articleForm';
import { useArticleFieldFocus } from './useArticleFieldFocus';
import { useArticleFieldModels, type ArticleFieldModels } from './useArticleFieldModels';
import { useArticleFormState } from './useArticleFormState';
import type { LockNoticeHandlers } from './useLockNotice';
import { useSaveArticle, type SaveTarget } from './useSaveArticle';

export type FormFeedback =
  | { readonly kind: 'none' }
  | { readonly kind: 'saved'; readonly revision: number }
  | { readonly kind: 'invalid'; readonly count: number }
  | { readonly kind: 'conflict'; readonly revision: number }
  | { readonly kind: 'unavailable' };

export type ArticleEditorModel = {
  readonly mode: SaveTarget['kind'];
  readonly fields: ArticleFieldModels;
  readonly feedback: FormFeedback;
  readonly savedAnnouncement: string;
  readonly isSaving: boolean;
  readonly isDirty: boolean;
  readonly submit: (event: FormEvent<HTMLFormElement>) => void;
  readonly reloadLatest: () => void;
};

export type ArticleEditorOptions = {
  readonly article: SellerArticle | null;
  readonly onCreated?: (article: SellerArticle) => void;
  readonly fetchLatest?: () => Promise<SellerArticle | null>;
  readonly lockNotice?: LockNoticeHandlers;
};

const noFeedback: FormFeedback = { kind: 'none' };

export function useArticleEditor({
  article,
  onCreated,
  fetchLatest,
  lockNotice,
}: ArticleEditorOptions): ArticleEditorModel {
  const { t } = useTranslation();
  const form = useArticleFormState(article);
  const { refs, focusField } = useArticleFieldFocus();
  const save = useSaveArticle();
  const [fieldErrors, setFieldErrors] = useState<ArticleFieldErrors>({});
  const [feedback, setFeedback] = useState<FormFeedback>(noFeedback);

  const onFieldEdited = useCallback((field: ArticleFormField) => {
    setFieldErrors((current) => withoutFieldError(current, field));
    setFeedback((current) => (current.kind === 'saved' ? noFeedback : current));
  }, []);
  const fields = useArticleFieldModels({ form, errors: fieldErrors, refs, onFieldEdited });

  const showFieldErrors = (errors: ArticleFieldErrors) => {
    setFieldErrors(errors);
    setFeedback({ kind: 'invalid', count: countFieldErrors(errors) });
    const firstField = firstInvalidField(errors);
    if (firstField !== null) {
      focusField(firstField);
    }
  };

  const showFailure = (error: unknown) => {
    const failure = toArticleFailure(error);
    switch (failure.kind) {
      case 'invalid':
        showFieldErrors(failure.errors);
        return;
      case 'conflict':
        setFeedback({ kind: 'conflict', revision: form.baseRevision ?? 1 });
        return;
      case 'locked':
        lockNotice?.show('not-modifiable');
        void fetchLatest?.();
        return;
      case 'not-found':
      case 'unavailable':
        setFeedback({ kind: 'unavailable' });
    }
  };

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (save.isPending) {
      return;
    }
    lockNotice?.clear();
    const conversion = toArticleDraft(form.values);
    if (conversion.status === 'invalid') {
      showFieldErrors(conversion.errors);
      return;
    }
    const target: SaveTarget =
      article === null || form.baseRevision === null
        ? { kind: 'create' }
        : { kind: 'update', id: article.id, revision: form.baseRevision };
    setFieldErrors({});
    setFeedback(noFeedback);
    save.mutate(
      { target, draft: conversion.draft },
      {
        onSuccess: (saved) => {
          form.reset(saved);
          setFeedback({ kind: 'saved', revision: saved.revision });
          if (target.kind === 'create') {
            onCreated?.(saved);
          }
        },
        onError: showFailure,
      },
    );
  };

  const reloadLatest = () => {
    void fetchLatest?.().then((latest) => {
      if (latest !== null) {
        form.reset(latest);
        setFieldErrors({});
        setFeedback(noFeedback);
      }
    });
  };

  return {
    mode: article === null ? 'create' : 'update',
    fields,
    feedback,
    savedAnnouncement:
      feedback.kind === 'saved' ? t('seller.form.saved', { revision: feedback.revision }) : '',
    isSaving: save.isPending,
    isDirty: form.isDirty,
    submit,
    reloadLatest,
  };
}
