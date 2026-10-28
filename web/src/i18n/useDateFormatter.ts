import { useCallback } from 'react';
import { localeTags } from './languages';
import { useLanguage } from './useLanguage';

export function useDateFormatter(): (date: Date) => string {
  const { language } = useLanguage();
  return useCallback(
    (date: Date) =>
      new Intl.DateTimeFormat(localeTags[language], { dateStyle: 'long' }).format(date),
    [language],
  );
}
