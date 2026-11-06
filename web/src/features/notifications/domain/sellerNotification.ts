import type { ArticleReason } from '../../../shared/articleReason';

export const notificationIssues = ['publie', 'refuse', 'a_verifier'] as const;

export type NotificationIssue = (typeof notificationIssues)[number];

export type SellerNotification = {
  readonly id: string;
  readonly articleId: string;
  readonly revision: number;
  readonly issue: NotificationIssue;
  readonly reason: ArticleReason | null;
  readonly decidedAt: Date;
};

export type NotificationsClient = {
  readonly listNotifications: (signal?: AbortSignal) => Promise<readonly SellerNotification[]>;
};

export const notificationsPageSize = 20;
