import { orderAccessFailureOf } from '../domain/orderProblems';

export const myOrdersPollingIntervalMs = 30_000;

export function pollUnlessAccessRefused(error: unknown): number | false {
  return orderAccessFailureOf(error) === null ? myOrdersPollingIntervalMs : false;
}
