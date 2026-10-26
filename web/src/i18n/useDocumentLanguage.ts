import { useEffect } from 'react';
import { useLanguage } from './useLanguage';

export function useDocumentLanguage(): void {
  const { language } = useLanguage();
  useEffect(() => {
    document.documentElement.lang = language;
  }, [language]);
}
