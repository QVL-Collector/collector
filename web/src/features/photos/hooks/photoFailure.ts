import { isApiError, type ApiError } from '../../../api/ApiError';
import type { PhotoRejection } from '../domain/photoInspection';
import { photoHttpStatuses, photoProblemTypes } from '../domain/photoProblems';

export type PhotoFailure =
  | PhotoRejection
  | 'limit-reached'
  | 'reserved'
  | 'locked'
  | 'rate-limited'
  | 'busy'
  | 'not-found'
  | 'unavailable';

const conflictFailures = new Map<string, PhotoFailure>([
  [photoProblemTypes.limitReached, 'limit-reached'],
  [photoProblemTypes.reserved, 'reserved'],
  [photoProblemTypes.notModifiable, 'locked'],
]);

const failuresByStatus = new Map<number, PhotoFailure>([
  [photoHttpStatuses.unsupportedMediaType, 'unsupported-type'],
  [photoHttpStatuses.payloadTooLarge, 'too-large'],
  [photoHttpStatuses.unprocessableContent, 'too-many-pixels'],
  [photoHttpStatuses.notFound, 'not-found'],
  [photoHttpStatuses.tooManyRequests, 'rate-limited'],
  [photoHttpStatuses.serviceUnavailable, 'busy'],
]);

function conflictFailure(problemType: string | null): PhotoFailure {
  const failure = problemType === null ? undefined : conflictFailures.get(problemType);
  return failure ?? 'locked';
}

function failureOf({ status, problem }: ApiError): PhotoFailure {
  if (status === photoHttpStatuses.conflict) {
    return conflictFailure(problem.type);
  }
  const failure = status === null ? undefined : failuresByStatus.get(status);
  return failure ?? 'unavailable';
}

export function toPhotoFailure(error: unknown): PhotoFailure {
  return isApiError(error) ? failureOf(error) : 'unavailable';
}
