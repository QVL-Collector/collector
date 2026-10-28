import { useMemo } from 'react';
import { localeTags } from './languages';
import { useLanguage } from './useLanguage';

const sampleDecimal = 1.5;
const fallbackSeparator = '.';

export function useDecimalSeparator(): string {
  const { language } = useLanguage();
  return useMemo(
    () =>
      new Intl.NumberFormat(localeTags[language])
        .formatToParts(sampleDecimal)
        .find((part) => part.type === 'decimal')?.value ?? fallbackSeparator,
    [language],
  );
}
