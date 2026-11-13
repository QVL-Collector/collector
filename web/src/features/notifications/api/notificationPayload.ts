import { articleReasons, type ArticleReason } from '../../../shared/articleReason';
import { parseDate, parseNullable, parseUuid } from '../../../shared/marketplacePayload';
import {
  expectArray,
  expectInteger,
  expectOneOf,
  expectRecord,
  InvalidPayloadError,
} from '../../../shared/payload';
import {
  notificationIssues,
  type NotificationIssue,
  type SellerNotification,
} from '../domain/sellerNotification';

function parseReason(value: unknown, path: string): ArticleReason | null {
  if (value === undefined) {
    throw new InvalidPayloadError(path);
  }
  return parseNullable(value, (reason) => expectOneOf(reason, articleReasons, path));
}

function expectReasonMatchesIssue(
  issue: NotificationIssue,
  reason: ArticleReason | null,
  path: string,
): void {
  const reasonExpected = issue !== 'publie';
  if (reasonExpected !== (reason !== null)) {
    throw new InvalidPayloadError(path);
  }
}

function parseNotification(value: unknown, path: string): SellerNotification {
  const notification = expectRecord(value, path);
  const issue = expectOneOf(notification.issue, notificationIssues, `${path}.issue`);
  const reason = parseReason(notification.motif, `${path}.motif`);
  expectReasonMatchesIssue(issue, reason, `${path}.motif`);
  return {
    id: parseUuid(notification.id, `${path}.id`),
    articleId: parseUuid(notification.articleId, `${path}.articleId`),
    revision: expectInteger(notification.revision, `${path}.revision`, 1),
    issue,
    reason,
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
