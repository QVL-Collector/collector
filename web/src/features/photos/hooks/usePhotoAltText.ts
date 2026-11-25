import { useTranslation } from 'react-i18next';

export function usePhotoAltText(articleTitle: string): string {
  const { t } = useTranslation();
  const title = articleTitle.trim();
  return title === '' ? t('photos.untitledAlt') : title;
}
