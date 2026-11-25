import { localeTags } from '../../../i18n/languages';
import { useLanguage } from '../../../i18n/useLanguage';
import { bytesPerMebibyte, pixelsPerMegapixel } from '../domain/articlePhoto';
import { usePhotoLimits } from './usePhotoLimits';

export type PhotoLimitLabels = {
  readonly maxSize: string;
  readonly maxPixels: string;
  readonly maxPerArticle: number;
};

const maximumFractionDigits = 1;

export function usePhotoLimitLabels(): PhotoLimitLabels {
  const { language } = useLanguage();
  const limits = usePhotoLimits();
  const format = new Intl.NumberFormat(localeTags[language], { maximumFractionDigits });
  return {
    maxSize: format.format(limits.maxBytes / bytesPerMebibyte),
    maxPixels: format.format(limits.maxPixels / pixelsPerMegapixel),
    maxPerArticle: limits.maxPerArticle,
  };
}
