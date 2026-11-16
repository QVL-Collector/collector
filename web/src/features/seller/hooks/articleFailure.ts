import { isApiError } from '../../../api/ApiError';
import { httpStatuses, sellerProblemTypes } from '../domain/sellerProblems';
import { countFieldErrors, toFieldErrors, type ArticleFieldErrors } from './articleForm';

export type ArticleFailure =
  | { readonly kind: 'invalid'; readonly errors: ArticleFieldErrors }
  | { readonly kind: 'conflict' }
  | { readonly kind: 'locked' }
  | { readonly kind: 'reserved' }
  | { readonly kind: 'not-found' }
  | { readonly kind: 'unavailable' };

type ConflictKind = Extract<ArticleFailure['kind'], 'conflict' | 'locked' | 'reserved'>;

const conflictKindsByProblemType: ReadonlyMap<string, ConflictKind> = new Map<string, ConflictKind>(
  [
    [sellerProblemTypes.notModifiable, 'locked'],
    [sellerProblemTypes.notSubmittable, 'locked'],
    [sellerProblemTypes.reserved, 'reserved'],
  ],
);

function conflictKindOf(problemType: string | null): ConflictKind {
  const kind = problemType === null ? undefined : conflictKindsByProblemType.get(problemType);
  return kind ?? 'conflict';
}

export function toArticleFailure(error: unknown): ArticleFailure {
  if (!isApiError(error)) {
    return { kind: 'unavailable' };
  }
  const { status, problem } = error;
  if (status === httpStatuses.conflict) {
    return { kind: conflictKindOf(problem.type) };
  }
  if (status === httpStatuses.notFound) {
    return { kind: 'not-found' };
  }
  const errors = toFieldErrors(problem.fieldProblems);
  return status === httpStatuses.badRequest && countFieldErrors(errors) > 0
    ? { kind: 'invalid', errors }
    : { kind: 'unavailable' };
}

export function isApiNotFound(error: unknown): boolean {
  return isApiError(error) && error.status === httpStatuses.notFound;
}
