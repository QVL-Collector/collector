import { ApiError } from '../../../api/ApiError';
import { waitForDemoLatency } from '../../../api/demoLatency';
import { isDemoArticleReserved, markDemoArticleReserved } from '../../../api/demoReservations';
import type { DemoScenario } from '../../../config/runtimeConfig';
import { demoArticles, type DemoArticlePayload } from '../../catalogue/api/demoArticles';
import type { Order, OrdersClient } from '../domain/order';
import { orderHttpStatuses } from '../domain/orderProblems';
import { parseOrder } from './orderPayload';

const commissionPercent = 5;
const percentBase = 100;
const paymentWindowMs = 30 * 60 * 1000;

type DemoAmount = { readonly montantCentimes: number; readonly devise: string };

function amount(montantCentimes: number, devise: string): DemoAmount {
  return { montantCentimes, devise };
}

function demoOrderPayload(article: DemoArticlePayload, now: Date): unknown {
  const { devise } = article.prix;
  const shipping = article.fraisDePort?.montantCentimes ?? 0;
  const total = article.prix.montantCentimes + shipping;
  const commission = Math.round((total * commissionPercent) / percentBase);
  return {
    id: crypto.randomUUID(),
    articleId: article.id,
    statut: 'en_attente_paiement',
    prix: article.prix,
    fraisDePort: amount(shipping, devise),
    totalPaye: amount(total, devise),
    commission: amount(commission, devise),
    montantVendeur: amount(total - commission, devise),
    echeance: new Date(now.getTime() + paymentWindowMs).toISOString(),
    creeLe: now.toISOString(),
  };
}

function findArticle(articleId: string): DemoArticlePayload {
  const article = demoArticles.find((candidate) => candidate.id === articleId);
  if (article === undefined) {
    throw new ApiError(orderHttpStatuses.notFound);
  }
  return article;
}

export function createDemoOrdersClient(scenario: DemoScenario): OrdersClient {
  const ordersById = new Map<string, Order>();
  const orderOf = (articleId: string) =>
    [...ordersById.values()].find((order) => order.articleId === articleId);
  const failIfUnavailable = () => {
    if (scenario === 'error') {
      throw new ApiError(orderHttpStatuses.serviceUnavailable);
    }
  };
  return {
    async reserveArticle(articleId) {
      await waitForDemoLatency();
      failIfUnavailable();
      const existing = orderOf(articleId);
      if (existing !== undefined) {
        return existing;
      }
      const article = findArticle(articleId);
      if (isDemoArticleReserved(articleId)) {
        throw new ApiError(orderHttpStatuses.conflict);
      }
      const order = parseOrder(demoOrderPayload(article, new Date()));
      ordersById.set(order.id, order);
      markDemoArticleReserved(articleId);
      return order;
    },
    async getOrder(id, signal) {
      await waitForDemoLatency(signal);
      failIfUnavailable();
      const order = ordersById.get(id);
      if (order === undefined) {
        throw new ApiError(orderHttpStatuses.notFound);
      }
      return order;
    },
  };
}
