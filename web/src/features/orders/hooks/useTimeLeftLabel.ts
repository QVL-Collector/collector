import { useTranslation } from 'react-i18next';
import { useRelativeTimeFormatter } from '../../../i18n/useRelativeTimeFormatter';
import type { Order } from '../domain/order';

const minuteMs = 60_000;

export function useTimeLeftLabel(order: Order, now: Date): string | null {
  const { t } = useTranslation();
  const formatRelative = useRelativeTimeFormatter();
  if (order.status === 'expiree') {
    return null;
  }
  const remainingMs = order.paymentDueAt.getTime() - now.getTime();
  if (remainingMs <= 0) {
    return t('orders.list.deadlineReached');
  }
  if (remainingMs < minuteMs) {
    return t('orders.list.lastMinute');
  }
  return t('orders.list.timeLeft', { relative: formatRelative(remainingMs) });
}
