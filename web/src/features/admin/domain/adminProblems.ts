export const adminProblemTypes = {
  invalidInput: 'urn:collector:probleme:saisie-invalide',
  staleRevision: 'urn:collector:probleme:revision-perimee',
  notUnderReview: 'urn:collector:probleme:article-non-a-verifier',
} as const;

export const adminHttpStatuses = {
  forbidden: 403,
  notFound: 404,
  conflict: 409,
  unprocessableContent: 422,
  serviceUnavailable: 503,
} as const;
