import type { UseQueryResult } from '@tanstack/react-query';
import type { Order } from '../domain/order';
import {
  isOrderNotFound,
  orderAccessFailureOf,
  type OrderAccessFailure,
} from '../domain/orderProblems';

export type OrderConfirmationView =
  | { readonly status: 'loading' }
  | { readonly status: 'not-found' }
  | { readonly status: OrderAccessFailure }
  | { readonly status: 'error' }
  | { readonly status: 'ready'; readonly order: Order };

export function toOrderConfirmationView(query: UseQueryResult<Order>): OrderConfirmationView {
  if (query.data !== undefined) {
    return { status: 'ready', order: query.data };
  }
  if (isOrderNotFound(query.error)) {
    return { status: 'not-found' };
  }
  const accessFailure = orderAccessFailureOf(query.error);
  if (accessFailure !== null) {
    return { status: accessFailure };
  }
  return query.isError ? { status: 'error' } : { status: 'loading' };
}
