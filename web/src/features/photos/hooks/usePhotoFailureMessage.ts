import { useTranslation } from 'react-i18next';
import type { PhotoFailure } from './photoFailure';
import { usePhotoLimitLabels } from './usePhotoLimitLabels';

export function usePhotoFailureMessage(failure: PhotoFailure | null): string | null {
  const { t } = useTranslation();
  const limits = usePhotoLimitLabels();
  switch (failure) {
    case null:
      return null;
    case 'unsupported-type':
      return t('photos.failures.unsupportedType');
    case 'too-large':
      return t('photos.failures.tooLarge', { max: limits.maxSize });
    case 'too-many-pixels':
      return t('photos.failures.tooManyPixels', { max: limits.maxPixels });
    case 'limit-reached':
      return t('photos.failures.limitReached', { count: limits.maxPerArticle });
    case 'reserved':
      return t('photos.failures.reserved');
    case 'locked':
      return t('photos.failures.locked');
    case 'rate-limited':
      return t('photos.failures.rateLimited');
    case 'busy':
      return t('photos.failures.busy');
    case 'not-found':
      return t('photos.failures.notFound');
    case 'unavailable':
      return t('photos.failures.unavailable');
  }
}
