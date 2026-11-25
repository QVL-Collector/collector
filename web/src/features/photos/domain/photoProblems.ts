export const photoProblemTypes = {
  limitReached: 'urn:collector:probleme:photos-limite-atteinte',
  tooLarge: 'urn:collector:probleme:photo-trop-volumineuse',
  unsupported: 'urn:collector:probleme:photo-non-prise-en-charge',
  excessiveDimensions: 'urn:collector:probleme:photo-dimensions-excessives',
  notModifiable: 'urn:collector:probleme:article-non-modifiable',
  reserved: 'urn:collector:probleme:article-reserve',
} as const;

export const photoHttpStatuses = {
  notFound: 404,
  conflict: 409,
  payloadTooLarge: 413,
  unsupportedMediaType: 415,
  unprocessableContent: 422,
  tooManyRequests: 429,
  serviceUnavailable: 503,
} as const;
