import type { HttpClient } from '../../../api/httpClient';
import type { OrdersClient } from '../domain/order';
import { parseOrder } from './orderPayload';

const ordersPath = '/v1/commandes';

function orderPath(id: string): string {
  return `${ordersPath}/${encodeURIComponent(id)}`;
}

export function createHttpOrdersClient(http: HttpClient): OrdersClient {
  return {
    async reserveArticle(articleId) {
      return parseOrder(await http.sendJson('POST', ordersPath, { articleId }));
    },
    async getOrder(id, signal) {
      return parseOrder(await http.getJson(orderPath(id), { signal }));
    },
  };
}
