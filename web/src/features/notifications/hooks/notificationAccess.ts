import { isApiError } from '../../../api/ApiError';
import { notificationHttpStatuses } from '../domain/notificationProblems';

export type AccessFailure = 'unauthenticated' | 'forbidden';

export function accessFailureOf(error: unknown): AccessFailure | null {
  if (!isApiError(error)) {
    return null;
  }
  if (error.status === notificationHttpStatuses.unauthorized) {
    return 'unauthenticated';
  }
  if (error.status === notificationHttpStatuses.forbidden) {
    return 'forbidden';
  }
  return null;
}
