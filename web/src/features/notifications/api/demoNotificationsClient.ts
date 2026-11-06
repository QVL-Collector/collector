import { ApiError } from '../../../api/ApiError';
import { waitForDemoLatency } from '../../../api/demoLatency';
import type { DemoScenario } from '../../../config/runtimeConfig';
import { notificationHttpStatuses } from '../domain/notificationProblems';
import { notificationsPageSize, type NotificationsClient } from '../domain/sellerNotification';
import { demoNotifications } from './demoNotifications';
import { parseNotificationPage } from './notificationPayload';

export function createDemoNotificationsClient(scenario: DemoScenario): NotificationsClient {
  const notifications = scenario === 'empty' ? [] : demoNotifications;
  return {
    async listNotifications(signal) {
      await waitForDemoLatency(signal);
      if (scenario === 'error') {
        throw new ApiError(notificationHttpStatuses.serviceUnavailable);
      }
      return parseNotificationPage({
        elements: notifications.slice(0, notificationsPageSize),
        page: 1,
        taille: notificationsPageSize,
        total: notifications.length,
      });
    },
  };
}
