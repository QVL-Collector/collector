import { isApiError } from '../../../api/ApiError';
import { adminHttpStatuses } from '../domain/adminProblems';

export type DecisionFailure = 'conflict' | 'forbidden' | 'unavailable';

const outdatedStatuses: readonly number[] = [
  adminHttpStatuses.conflict,
  adminHttpStatuses.notFound,
];

export function toDecisionFailure(error: unknown): DecisionFailure {
  if (!isApiError(error) || error.status === null) {
    return 'unavailable';
  }
  if (outdatedStatuses.includes(error.status)) {
    return 'conflict';
  }
  return error.status === adminHttpStatuses.forbidden ? 'forbidden' : 'unavailable';
}
