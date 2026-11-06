import { articleReasons, type ArticleReason } from '../../../shared/articleReason';
import { parseDate, parseNullable, parseUuid } from '../../../shared/marketplacePayload';
import { expectArray, expectInteger, expectOneOf, expectRecord } from '../../../shared/payload';
import { notificationIssues, type SellerNotification } from '../domain/sellerNotification';

function parseReason(value: unknown, path: string): ArticleReason | null {
  return parseNullable(value ?? null, (reason) => expectOneOf(reason, articleReasons, path));
}

export function parseNotification(value: unknown, path: string): SellerNotification {
  const notification = expectRecord(value, path);
  return {
    id: parseUuid(notification.id, `${path}.id`),
    articleId: parseUuid(notification.articleId, `${path}.articleId`),
    revision: expectInteger(notification.revision, `${path}.revision`, 1),
    issue: expectOneOf(notification.issue, notificationIssues, `${path}.issue`),
    reason: parseReason(notification.motif, `${path}.motif`),
    decidedAt: parseDate(notification.decideLe, `${path}.decideLe`),
  };
}

export function parseNotificationPage(payload: unknown): readonly SellerNotification[] {
  const root = expectRecord(payload, 'notifications');
  expectInteger(root.page, 'page', 1);
  expectInteger(root.taille, 'taille', 1);
  expectInteger(root.total, 'total', 0);
  return expectArray(root.elements, 'elements').map((item, index) =>
    parseNotification(item, `elements[${String(index)}]`),
  );
}
