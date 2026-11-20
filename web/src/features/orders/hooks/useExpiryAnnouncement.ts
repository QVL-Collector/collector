import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { Order, OrderStatus } from '../domain/order';

type KnownStatuses = ReadonlyMap<string, OrderStatus>;

function statusesOf(orders: readonly Order[]): KnownStatuses {
  return new Map(orders.map((order): [string, OrderStatus] => [order.id, order.status]));
}

function newlyExpired(orders: readonly Order[], previous: KnownStatuses): readonly Order[] {
  return orders.filter(
    (order) => order.status === 'expiree' && previous.get(order.id) === 'en_attente_paiement',
  );
}

export function useExpiryAnnouncement(orders: readonly Order[] | undefined): string {
  const { t } = useTranslation();
  const knownStatuses = useRef<KnownStatuses | null>(null);
  const [announcement, setAnnouncement] = useState('');
  useEffect(() => {
    if (orders === undefined) {
      return;
    }
    const previous = knownStatuses.current;
    knownStatuses.current = statusesOf(orders);
    const expired = previous === null ? [] : newlyExpired(orders, previous);
    if (expired.length > 0) {
      setAnnouncement(
        expired.map((order) => t('orders.list.expired', { title: order.title })).join(' '),
      );
    }
  }, [orders, t]);
  return announcement;
}
