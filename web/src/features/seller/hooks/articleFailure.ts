import { isApiError } from '../../../api/ApiError';
import { httpStatuses, sellerProblemTypes } from '../domain/sellerProblems';
import { countFieldErrors, toFieldErrors, type ArticleFieldErrors } from './articleForm';

export type ArticleFailure =
  | { readonly kind: 'invalid'; readonly errors: ArticleFieldErrors }
  | { readonly kind: 'conflict' }
  | { readonly kind: 'locked' }
  | { readonly kind: 'not-found' }
  | { readonly kind: 'unavailable' };

const lockedProblemTypes: readonly string[] = [
  sellerProblemTypes.notModifiable,
  sellerProblemTypes.notSubmittable,
];

export function toArticleFailure(error: unknown): ArticleFailure {
  if (!isApiError(error)) {
    return { kind: 'unavailable' };
  }
  const { status, problem } = error;
  if (status === httpStatuses.conflict) {
    const isLocked = problem.type !== null && lockedProblemTypes.includes(problem.type);
    return { kind: isLocked ? 'locked' : 'conflict' };
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
