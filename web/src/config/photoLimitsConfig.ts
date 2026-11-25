import {
  defaultPhotoLimits,
  maxPhotosPerArticleCeiling,
  type PhotoLimits,
} from '../features/photos/domain/articlePhoto';
import { expectInteger, expectRecord, InvalidPayloadError } from '../shared/payload';

function optionalPositiveInteger(value: unknown, path: string, fallback: number): number {
  return value === undefined ? fallback : expectInteger(value, path, 1);
}

function parseMaxPerArticle(value: unknown): number {
  const count = optionalPositiveInteger(
    value,
    'photos.maxPerArticle',
    defaultPhotoLimits.maxPerArticle,
  );
  if (count > maxPhotosPerArticleCeiling) {
    throw new InvalidPayloadError('photos.maxPerArticle');
  }
  return count;
}

export function parsePhotoLimits(value: unknown): PhotoLimits {
  if (value === undefined) {
    return defaultPhotoLimits;
  }
  const photos = expectRecord(value, 'photos');
  return {
    maxBytes: optionalPositiveInteger(
      photos.maxBytes,
      'photos.maxBytes',
      defaultPhotoLimits.maxBytes,
    ),
    maxPixels: optionalPositiveInteger(
      photos.maxPixels,
      'photos.maxPixels',
      defaultPhotoLimits.maxPixels,
    ),
    maxPerArticle: parseMaxPerArticle(photos.maxPerArticle),
  };
}
