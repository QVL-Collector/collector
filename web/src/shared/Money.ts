export type Money = {
  readonly amountInCents: number;
  readonly currency: string;
};

const currencyCodePattern = /^[A-Z]{3}$/;

export function isCurrencyCode(value: string): boolean {
  return currencyCodePattern.test(value);
}
