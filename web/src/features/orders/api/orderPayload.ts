import { parseDate, parseMoney, parseUuid } from '../../../shared/marketplacePayload';
import {
  expectArray,
  expectInteger,
  expectNonEmptyString,
  expectOneOf,
  expectRecord,
} from '../../../shared/payload';
import { orderStatuses, type Order, type OrderPage } from '../domain/order';

export function parseOrder(payload: unknown, path = 'commande'): Order {
  const order = expectRecord(payload, path);
  return {
    id: parseUuid(order.id, `${path}.id`),
    articleId: expectNonEmptyString(order.articleId, `${path}.articleId`),
    title: expectNonEmptyString(order.titre, `${path}.titre`),
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

export function parseOrderPage(payload: unknown): OrderPage {
  const root = expectRecord(payload, 'commandes');
  return {
    orders: expectArray(root.elements, 'elements').map((item, index) =>
      parseOrder(item, `elements[${String(index)}]`),
    ),
    page: expectInteger(root.page, 'page', 1),
    pageSize: expectInteger(root.taille, 'taille', 1),
    totalItems: expectInteger(root.total, 'total', 0),
  };
}
