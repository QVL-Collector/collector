import type { Money } from '../../../shared/Money';

export const orderStatuses = ['en_attente_paiement', 'expiree'] as const;

export type OrderStatus = (typeof orderStatuses)[number];

export type Order = {
  readonly id: string;
  readonly articleId: string;
  readonly status: OrderStatus;
  readonly price: Money;
  readonly shippingCost: Money;
  readonly totalPaid: Money;
  readonly commission: Money;
  readonly sellerAmount: Money;
  readonly paymentDueAt: Date;
  readonly createdAt: Date;
};

export type OrdersClient = {
  readonly reserveArticle: (articleId: string) => Promise<Order>;
  readonly getOrder: (id: string, signal?: AbortSignal) => Promise<Order>;
};
