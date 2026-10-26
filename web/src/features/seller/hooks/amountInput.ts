import type { FieldProblemCode } from '../domain/sellerProblems';

export type AmountInput =
  | { readonly status: 'empty' }
  | { readonly status: 'valid'; readonly amountInCents: number }
  | { readonly status: 'invalid'; readonly code: FieldProblemCode };

const centsPerUnit = 100;
const centsDigits = 2;
const amountPattern = /^(\d+)(?:[.,](\d{1,2}))?$/;
const whitespacePattern = /\s/g;

export function parseAmountInput(text: string): AmountInput {
  const compact = text.replace(whitespacePattern, '');
  if (compact === '') {
    return { status: 'empty' };
  }
  if (compact.startsWith('-')) {
    return { status: 'invalid', code: 'negatif' };
  }
  const match = amountPattern.exec(compact);
  if (match === null) {
    return { status: 'invalid', code: 'invalide' };
  }
  const [, units = '', cents = ''] = match;
  const amountInCents = Number(units) * centsPerUnit + Number(cents.padEnd(centsDigits, '0'));
  return Number.isSafeInteger(amountInCents)
    ? { status: 'valid', amountInCents }
    : { status: 'invalid', code: 'hors_limites' };
}

export function formatAmountInput(amountInCents: number, decimalSeparator: string): string {
  const units = Math.floor(amountInCents / centsPerUnit);
  const cents = amountInCents % centsPerUnit;
  return cents === 0
    ? String(units)
    : `${String(units)}${decimalSeparator}${String(cents).padStart(centsDigits, '0')}`;
}
