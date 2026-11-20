import type { UseQueryResult } from '@tanstack/react-query';
import type { Order, OrderPage } from '../domain/order';
import { orderAccessFailureOf, type OrderAccessFailure } from '../domain/orderProblems';

export type MyOrdersView =
  | { readonly status: 'loading' }
  | { readonly status: OrderAccessFailure }
  | { readonly status: 'error' }
  | { readonly status: 'empty' }
  | {
      readonly status: 'ready';
      readonly orders: readonly Order[];
      readonly page: number;
      readonly pageCount: number;
      readonly totalItems: number;
      readonly isRefreshing: boolean;
    };

export function toMyOrdersView(
  query: UseQueryResult<OrderPage>,
  requestedPage: number,
): MyOrdersView {
  const accessFailure = orderAccessFailureOf(query.error);
  if (accessFailure !== null) {
    return { status: accessFailure };
  }
  if (query.data === undefined) {
    return query.isError ? { status: 'error' } : { status: 'loading' };
  }
  const { orders, totalItems, pageSize } = query.data;
  if (totalItems === 0) {
    return { status: 'empty' };
  }
  return {
    status: 'ready',
    orders,
    page: requestedPage,
    pageCount: Math.max(1, Math.ceil(totalItems / pageSize)),
    totalItems,
    isRefreshing: query.isPlaceholderData,
  };
}
