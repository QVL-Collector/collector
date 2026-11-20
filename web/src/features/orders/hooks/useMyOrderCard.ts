import { useTranslation } from 'react-i18next';
import type { Order } from '../domain/order';
import { orderPath } from '../ordersPaths';
import { useOrderSummary, type OrderSummary } from './useOrderSummary';
import { useTimeLeftLabel } from './useTimeLeftLabel';

export type MyOrderCardModel = Pick<
  OrderSummary,
  | 'articleTitle'
  | 'articlePath'
  | 'status'
  | 'statusLabel'
  | 'totalPaidLabel'
  | 'deadlineTitle'
  | 'paymentDueLabel'
  | 'paymentDueDateTime'
  | 'createdLabel'
  | 'createdDateTime'
> & {
  readonly totalTitle: string;
  readonly timeLeftLabel: string | null;
  readonly detailPath: string;
  readonly detailLabel: string;
  readonly detailAccessibleLabel: string;
};

export function useMyOrderCard(order: Order, now: Date): MyOrderCardModel {
  const { t } = useTranslation();
  const summary = useOrderSummary(order);
  const timeLeftLabel = useTimeLeftLabel(order, now);
  return {
    articleTitle: summary.articleTitle,
    articlePath: summary.articlePath,
    status: summary.status,
    statusLabel: summary.statusLabel,
    totalPaidLabel: summary.totalPaidLabel,
    deadlineTitle: summary.deadlineTitle,
    paymentDueLabel: summary.paymentDueLabel,
    paymentDueDateTime: summary.paymentDueDateTime,
    createdLabel: summary.createdLabel,
    createdDateTime: summary.createdDateTime,
    totalTitle: t('orders.list.total'),
    timeLeftLabel,
    detailPath: orderPath(order.id),
    detailLabel: t('orders.list.viewDetail'),
    detailAccessibleLabel: t('orders.list.viewDetailLabel', { title: order.title }),
  };
}
