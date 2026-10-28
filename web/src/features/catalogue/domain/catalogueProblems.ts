import { isApiError } from '../../../api/ApiError';

export const unknownCategoryCode = 'categorie_inconnue';

const notFoundStatuses: readonly number[] = [400, 404];

const badRequestStatus = 400;

export function isUnknownCategory(error: unknown): boolean {
  return (
    isApiError(error) &&
    error.status === badRequestStatus &&
    error.problem.fieldProblems.some((problem) => problem.code === unknownCategoryCode)
  );
}

export function isArticleNotFound(error: unknown): boolean {
  return isApiError(error) && error.status !== null && notFoundStatuses.includes(error.status);
}
