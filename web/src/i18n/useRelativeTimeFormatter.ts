import { useCallback } from 'react';
import { localeTags } from './languages';
import { useLanguage } from './useLanguage';

const minuteMs = 60_000;
const minutesPerHour = 60;

export function useRelativeTimeFormatter(): (durationMs: number) => string {
  const { language } = useLanguage();
  return useCallback(
    (durationMs: number) => {
      const formatter = new Intl.RelativeTimeFormat(localeTags[language], { numeric: 'always' });
      const minutes = Math.ceil(durationMs / minuteMs);
      return minutes < minutesPerHour
        ? formatter.format(minutes, 'minute')
        : formatter.format(Math.floor(minutes / minutesPerHour), 'hour');
    },
    [language],
  );
}
