import { useCallback } from 'react';
import type { Money } from '../shared/Money';
import { localeTags } from './languages';
import { useLanguage } from './useLanguage';

const centsPerUnit = 100;

export function useMoneyFormatter(): (money: Money) => string {
  const { language } = useLanguage();
  return useCallback(
    (money: Money) =>
      new Intl.NumberFormat(localeTags[language], {
        style: 'currency',
        currency: money.currency,
      }).format(money.amountInCents / centsPerUnit),
    [language],
  );
}
