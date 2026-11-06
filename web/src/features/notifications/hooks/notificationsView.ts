import type { UseQueryResult } from '@tanstack/react-query';
import type { SellerNotification } from '../domain/sellerNotification';
import { accessFailureOf, type AccessFailure } from './notificationAccess';

export type NotificationsView =
  | { readonly status: 'loading' }
  | { readonly status: AccessFailure }
  | { readonly status: 'error' }
  | { readonly status: 'empty' }
  | { readonly status: 'ready'; readonly notifications: readonly SellerNotification[] };

function byMostRecent(left: SellerNotification, right: SellerNotification): number {
  return right.decidedAt.getTime() - left.decidedAt.getTime();
}

export function toNotificationsView(
  query: UseQueryResult<readonly SellerNotification[]>,
): NotificationsView {
  const accessFailure = accessFailureOf(query.error);
  if (accessFailure !== null) {
    return { status: accessFailure };
  }
  if (query.data === undefined) {
    return query.isError ? { status: 'error' } : { status: 'loading' };
  }
  if (query.data.length === 0) {
    return { status: 'empty' };
  }
  return { status: 'ready', notifications: [...query.data].sort(byMostRecent) };
}
