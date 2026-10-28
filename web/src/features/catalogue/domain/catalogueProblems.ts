import { isApiError } from '../../../api/ApiError';

export const unknownCategoryCode = 'categorie_inconnue';

const badRequestStatus = 400;

const notFoundStatus = 404;

export function isUnknownCategory(error: unknown): boolean {
  return (
    isApiError(error) &&
    error.status === badRequestStatus &&
    error.problem.fieldProblems.some((problem) => problem.code === unknownCategoryCode)
  );
}

export function isArticleNotFound(error: unknown): boolean {
  return isApiError(error) && error.status === notFoundStatus;
}
