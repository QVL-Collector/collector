import { isApiError, type ApiError } from '../../../api/ApiError';
import { orderHttpStatuses, orderProblemTypes } from '../domain/orderProblems';

export type ReservationFailure =
  | 'already-reserved'
  | 'reservation-already-active'
  | 'not-reservable'
  | 'session-expired'
  | 'forbidden'
  | 'rate-limited'
  | 'unavailable';

const failuresByStatus: ReadonlyMap<number, ReservationFailure> = new Map<
  number,
  ReservationFailure
>([
  [orderHttpStatuses.notFound, 'not-reservable'],
  [orderHttpStatuses.unprocessable, 'not-reservable'],
  [orderHttpStatuses.unauthorized, 'session-expired'],
  [orderHttpStatuses.forbidden, 'forbidden'],
  [orderHttpStatuses.tooManyRequests, 'rate-limited'],
]);

function conflictFailureOf(error: ApiError): ReservationFailure {
  return error.problem.type === orderProblemTypes.reservationAlreadyActive
    ? 'reservation-already-active'
    : 'already-reserved';
}

export function toReservationFailure(error: unknown): ReservationFailure {
  if (!isApiError(error) || error.status === null) {
    return 'unavailable';
  }
  if (error.status === orderHttpStatuses.conflict) {
    return conflictFailureOf(error);
  }
  return failuresByStatus.get(error.status) ?? 'unavailable';
}

export function blocksReservation(failure: ReservationFailure | null): boolean {
  return failure === 'already-reserved' || failure === 'not-reservable';
}
