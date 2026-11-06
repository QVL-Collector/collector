import { useTranslation } from 'react-i18next';
import { useArticleReasonLabel } from '../../../i18n/useArticleReasonLabel';
import { useDateTimeFormatter } from '../../../i18n/useDateTimeFormatter';
import { articlePath } from '../../seller/sellerPaths';
import type { NotificationIssue, SellerNotification } from '../domain/sellerNotification';

export type NotificationCardModel = {
  readonly path: string;
  readonly issue: NotificationIssue;
  readonly issueLabel: string;
  readonly title: string;
  readonly description: string;
  readonly reasonLabel: string | null;
  readonly decidedLabel: string;
  readonly decidedDateTime: string;
  readonly revisionLabel: string;
};

export function useNotificationCard(notification: SellerNotification): NotificationCardModel {
  const { t } = useTranslation();
  const reasonLabelOf = useArticleReasonLabel();
  const formatDateTime = useDateTimeFormatter();
  const { issue, reason, revision, decidedAt } = notification;
  return {
    path: articlePath(notification.articleId),
    issue,
    issueLabel: t(`seller.states.${issue}`),
    title: t(`notifications.issues.${issue}.title`),
    description: t(`notifications.issues.${issue}.description`),
    reasonLabel: reason === null ? null : t('seller.reason', { reason: reasonLabelOf(reason) }),
    decidedLabel: t('notifications.decidedAt', { date: formatDateTime(decidedAt) }),
    decidedDateTime: decidedAt.toISOString(),
    revisionLabel: t('notifications.revision', { revision }),
  };
}
