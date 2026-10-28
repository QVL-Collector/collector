import { isCurrencyCode, type Money } from './Money';
import {
  expectArray,
  expectInteger,
  expectNonEmptyString,
  expectRecord,
  expectString,
  InvalidPayloadError,
} from './payload';

export type CategoryPayload = {
  readonly id: string;
  readonly label: string;
};

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export function parseNullable<T>(value: unknown, parse: (present: unknown) => T): T | null {
  return value === null ? null : parse(value);
}

export function parseDate(value: unknown, path: string): Date {
  const date = new Date(expectNonEmptyString(value, path));
  if (Number.isNaN(date.getTime())) {
    throw new InvalidPayloadError(path);
  }
  return date;
}

export function parseUuid(value: unknown, path: string): string {
  const id = expectString(value, path);
  if (!uuidPattern.test(id)) {
    throw new InvalidPayloadError(path);
  }
  return id;
}

export function parseMoney(value: unknown, path: string): Money {
  const money = expectRecord(value, path);
  const currency = expectNonEmptyString(money.devise, `${path}.devise`);
  if (!isCurrencyCode(currency)) {
    throw new InvalidPayloadError(`${path}.devise`);
  }
  return {
    amountInCents: expectInteger(money.montantCentimes, `${path}.montantCentimes`, 0),
    currency,
  };
}

export function parseCategory(value: unknown, path: string): CategoryPayload {
  const category = expectRecord(value, path);
  return {
    id: expectNonEmptyString(category.id, `${path}.id`),
    label: expectNonEmptyString(category.libelle, `${path}.libelle`),
  };
}

export function parseCategories(payload: unknown): readonly CategoryPayload[] {
  const root = expectRecord(payload, 'categories');
  return expectArray(root.elements, 'elements').map((item, index) =>
    parseCategory(item, `elements[${String(index)}]`),
  );
}
