import { isCurrencyCode, type Money } from './Money';
import { expectInteger, expectNonEmptyString, expectRecord, InvalidPayloadError } from './payload';

export type CategoryPayload = {
  readonly id: string;
  readonly label: string;
};

export function parseNullable<T>(value: unknown, parse: (present: unknown) => T): T | null {
  return value === null ? null : parse(value);
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
