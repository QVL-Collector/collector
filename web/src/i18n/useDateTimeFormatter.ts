import { useCallback } from 'react';
import { localeTags } from './languages';
import { useLanguage } from './useLanguage';

const dateTimeOptions: Intl.DateTimeFormatOptions = { dateStyle: 'long', timeStyle: 'short' };

export function useDateTimeFormatter(): (date: Date) => string {
  const { language } = useLanguage();
  return useCallback(
    (date: Date) => new Intl.DateTimeFormat(localeTags[language], dateTimeOptions).format(date),
    [language],
  );
}
