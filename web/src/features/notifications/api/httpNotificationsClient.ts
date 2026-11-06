import type { HttpClient } from '../../../api/httpClient';
import { notificationsPageSize, type NotificationsClient } from '../domain/sellerNotification';
import { parseNotificationPage } from './notificationPayload';

const notificationsPath = '/v1/notifications';
const firstPage = 1;

export function createHttpNotificationsClient(http: HttpClient): NotificationsClient {
  return {
    async listNotifications(signal) {
      const payload = await http.getJson(notificationsPath, {
        query: { page: String(firstPage), taille: String(notificationsPageSize) },
        signal,
      });
      return parseNotificationPage(payload);
    },
  };
}
