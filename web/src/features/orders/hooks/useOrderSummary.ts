import { useTranslation } from 'react-i18next';
import { useDateTimeFormatter } from '../../../i18n/useDateTimeFormatter';
import { useMoneyFormatter } from '../../../i18n/useMoneyFormatter';
import { publishedArticlePath } from '../../catalogue/cataloguePaths';
import type { Order, OrderStatus } from '../domain/order';

export type OrderSummary = {
  readonly status: OrderStatus;
  readonly statusLabel: string;
  readonly deadlineTitle: string;
  readonly deadlineHint: string;
  readonly paymentDueLabel: string;
  readonly paymentDueDateTime: string;
  readonly createdLabel: string;
  readonly createdDateTime: string;
  readonly priceLabel: string;
  readonly shippingLabel: string;
  readonly totalPaidLabel: string;
  readonly commissionLabel: string;
  readonly sellerAmountLabel: string;
  readonly articlePath: string;
};

export function useOrderSummary(order: Order): OrderSummary {
  const { t } = useTranslation();
  const formatMoney = useMoneyFormatter();
  const formatDateTime = useDateTimeFormatter();
  return {
    status: order.status,
    statusLabel: t(`orders.statuses.${order.status}`),
    deadlineTitle: t(`orders.deadlines.${order.status}.title`),
    deadlineHint: t(`orders.deadlines.${order.status}.hint`),
    paymentDueLabel: formatDateTime(order.paymentDueAt),
    paymentDueDateTime: order.paymentDueAt.toISOString(),
    createdLabel: t('orders.createdAt', { date: formatDateTime(order.createdAt) }),
    createdDateTime: order.createdAt.toISOString(),
    priceLabel: formatMoney(order.price),
    shippingLabel: formatMoney(order.shippingCost),
    totalPaidLabel: formatMoney(order.totalPaid),
    commissionLabel: formatMoney(order.commission),
    sellerAmountLabel: formatMoney(order.sellerAmount),
    articlePath: publishedArticlePath(order.articleId),
  };
}
