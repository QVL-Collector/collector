import { ApiError } from '../../../api/ApiError';
import { waitForDemoLatency } from '../../../api/demoLatency';
import { isDemoArticleReserved, markDemoArticleReserved } from '../../../api/demoReservations';
import type { DemoScenario } from '../../../config/runtimeConfig';
import { demoArticles, type DemoArticlePayload } from '../../catalogue/api/demoArticles';
import type { Order, OrdersClient, OrderStatus } from '../domain/order';
import { orderHttpStatuses } from '../domain/orderProblems';
import { parseOrder } from './orderPayload';

const commissionPercent = 5;
const percentBase = 100;
const paymentWindowMs = 30 * 60 * 1000;

type DemoAmount = { readonly montantCentimes: number; readonly devise: string };

type DemoOrderRecord = {
  readonly id: string;
  readonly articleId: string;
  readonly prix: DemoAmount;
  readonly fraisDePort: DemoAmount;
  readonly totalPaye: DemoAmount;
  readonly commission: DemoAmount;
  readonly montantVendeur: DemoAmount;
  readonly echeance: Date;
  readonly creeLe: Date;
};

function amount(montantCentimes: number, devise: string): DemoAmount {
  return { montantCentimes, devise };
}

function demoOrderRecord(article: DemoArticlePayload, now: Date): DemoOrderRecord {
  const { devise } = article.prix;
  const shipping = article.fraisDePort?.montantCentimes ?? 0;
  const total = article.prix.montantCentimes + shipping;
  const commission = Math.round((total * commissionPercent) / percentBase);
  return {
    id: crypto.randomUUID(),
    articleId: article.id,
    prix: article.prix,
    fraisDePort: amount(shipping, devise),
    totalPaye: amount(total, devise),
    commission: amount(commission, devise),
    montantVendeur: amount(total - commission, devise),
    echeance: new Date(now.getTime() + paymentWindowMs),
    creeLe: now,
  };
}

function isExpired(record: DemoOrderRecord, now: Date): boolean {
  return now.getTime() >= record.echeance.getTime();
}

function statusOf(record: DemoOrderRecord, now: Date): OrderStatus {
  return isExpired(record, now) ? 'expiree' : 'en_attente_paiement';
}

function toOrder(record: DemoOrderRecord, now: Date): Order {
  return parseOrder({
    ...record,
    statut: statusOf(record, now),
    echeance: record.echeance.toISOString(),
    creeLe: record.creeLe.toISOString(),
  });
}

function findArticle(articleId: string): DemoArticlePayload {
  const article = demoArticles.find((candidate) => candidate.id === articleId);
  if (article === undefined) {
    throw new ApiError(orderHttpStatuses.notFound);
  }
  return article;
}

export function createDemoOrdersClient(scenario: DemoScenario): OrdersClient {
  const recordsById = new Map<string, DemoOrderRecord>();
  const activeRecordOf = (articleId: string, now: Date) =>
    [...recordsById.values()].find(
      (record) => record.articleId === articleId && !isExpired(record, now),
    );
  const failIfUnavailable = () => {
    if (scenario === 'error') {
      throw new ApiError(orderHttpStatuses.serviceUnavailable);
    }
  };
  return {
    async reserveArticle(articleId) {
      await waitForDemoLatency();
      failIfUnavailable();
      const now = new Date();
      const existing = activeRecordOf(articleId, now);
      if (existing !== undefined) {
        return toOrder(existing, now);
      }
      const article = findArticle(articleId);
      if (isDemoArticleReserved(articleId, now)) {
        throw new ApiError(orderHttpStatuses.conflict);
      }
      const record = demoOrderRecord(article, now);
      recordsById.set(record.id, record);
      markDemoArticleReserved(articleId, record.echeance);
      return toOrder(record, now);
    },
    async getOrder(id, signal) {
      await waitForDemoLatency(signal);
      failIfUnavailable();
      const record = recordsById.get(id);
      if (record === undefined) {
        throw new ApiError(orderHttpStatuses.notFound);
      }
      return toOrder(record, new Date());
    },
  };
}
