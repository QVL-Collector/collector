import { parseDate, parseMoney, parseUuid } from '../../../shared/marketplacePayload';
import { expectNonEmptyString, expectOneOf, expectRecord } from '../../../shared/payload';
import { orderStatuses, type Order } from '../domain/order';

export function parseOrder(payload: unknown): Order {
  const path = 'commande';
  const order = expectRecord(payload, path);
  return {
    id: parseUuid(order.id, `${path}.id`),
    articleId: expectNonEmptyString(order.articleId, `${path}.articleId`),
    status: expectOneOf(order.statut, orderStatuses, `${path}.statut`),
    price: parseMoney(order.prix, `${path}.prix`),
    shippingCost: parseMoney(order.fraisDePort, `${path}.fraisDePort`),
    totalPaid: parseMoney(order.totalPaye, `${path}.totalPaye`),
    commission: parseMoney(order.commission, `${path}.commission`),
    sellerAmount: parseMoney(order.montantVendeur, `${path}.montantVendeur`),
    paymentDueAt: parseDate(order.echeance, `${path}.echeance`),
    createdAt: parseDate(order.creeLe, `${path}.creeLe`),
  };
}
