import { isApiError } from '../../../api/ApiError';

export const orderHttpStatuses = {
  unauthorized: 401,
  forbidden: 403,
  notFound: 404,
  conflict: 409,
  unprocessable: 422,
  tooManyRequests: 429,
  serviceUnavailable: 503,
} as const;

export const orderProblemTypes = {
  articleAlreadyReserved: 'urn:collector:probleme:article-deja-reserve',
  reservationAlreadyActive: 'urn:collector:probleme:reservation-deja-active',
} as const;

export type OrderAccessFailure = 'unauthenticated' | 'forbidden';

const accessFailuresByStatus: ReadonlyMap<number, OrderAccessFailure> = new Map<
  number,
  OrderAccessFailure
>([
  [orderHttpStatuses.unauthorized, 'unauthenticated'],
  [orderHttpStatuses.forbidden, 'forbidden'],
]);

export function orderAccessFailureOf(error: unknown): OrderAccessFailure | null {
  if (!isApiError(error) || error.status === null) {
    return null;
  }
  return accessFailuresByStatus.get(error.status) ?? null;
}

export function isOrderNotFound(error: unknown): boolean {
  return isApiError(error) && error.status === orderHttpStatuses.notFound;
}
