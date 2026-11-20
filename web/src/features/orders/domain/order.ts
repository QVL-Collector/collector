import type { Money } from '../../../shared/Money';

export const orderStatuses = ['en_attente_paiement', 'expiree'] as const;

export type OrderStatus = (typeof orderStatuses)[number];

export const ordersPageSize = 20;

export type Order = {
  readonly id: string;
  readonly articleId: string;
  readonly title: string;
  readonly status: OrderStatus;
  readonly price: Money;
  readonly shippingCost: Money;
  readonly totalPaid: Money;
  readonly commission: Money;
  readonly sellerAmount: Money;
  readonly paymentDueAt: Date;
  readonly createdAt: Date;
};

export type OrderPageRequest = {
  readonly page: number;
  readonly pageSize: number;
};

export type OrderPage = {
  readonly orders: readonly Order[];
  readonly page: number;
  readonly pageSize: number;
  readonly totalItems: number;
};

export type OrdersClient = {
  readonly reserveArticle: (articleId: string) => Promise<Order>;
  readonly getOrder: (id: string, signal?: AbortSignal) => Promise<Order>;
  readonly listMyOrders: (request: OrderPageRequest, signal?: AbortSignal) => Promise<OrderPage>;
};
