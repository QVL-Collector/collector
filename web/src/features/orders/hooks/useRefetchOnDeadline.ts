import { useEffect, useRef } from 'react';
import type { Order } from '../domain/order';

function deadlineKeyOf(order: Order): string {
  return `${order.id}:${String(order.paymentDueAt.getTime())}`;
}

function hasReachedDeadline(order: Order, now: Date): boolean {
  return order.status === 'en_attente_paiement' && order.paymentDueAt.getTime() <= now.getTime();
}

function reachedDeadlineKeys(orders: readonly Order[], now: Date): readonly string[] {
  return orders.filter((order) => hasReachedDeadline(order, now)).map(deadlineKeyOf);
}

export function useRefetchOnDeadline(
  orders: readonly Order[] | undefined,
  now: Date,
  refetch: () => void,
): void {
  const handledKeys = useRef(new Set<string>());
  useEffect(() => {
    const newKeys = reachedDeadlineKeys(orders ?? [], now).filter(
      (key) => !handledKeys.current.has(key),
    );
    if (newKeys.length === 0) {
      return;
    }
    for (const key of newKeys) {
      handledKeys.current.add(key);
    }
    refetch();
  }, [orders, now, refetch]);
}
