import { accessFailureOf } from './notificationAccess';

export const notificationsPollingIntervalMs = 5000;

export function pollUnlessAccessRefused(error: unknown): number | false {
  return accessFailureOf(error) === null ? notificationsPollingIntervalMs : false;
}
