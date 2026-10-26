export const sellerProblemTypes = {
  invalidInput: 'urn:collector:probleme:saisie-invalide',
  staleRevision: 'urn:collector:probleme:revision-perimee',
  notModifiable: 'urn:collector:probleme:article-non-modifiable',
  notSubmittable: 'urn:collector:probleme:article-non-soumettable',
} as const;

export const fieldProblemCodes = [
  'obligatoire',
  'invalide',
  'hors_limites',
  'trop_long',
  'caractere_interdit',
  'negatif',
  'au_dela_du_plafond',
  'devise_non_prise_en_charge',
  'categorie_inconnue',
] as const;

export type FieldProblemCode = (typeof fieldProblemCodes)[number];

export const httpStatuses = {
  badRequest: 400,
  notFound: 404,
  conflict: 409,
  serviceUnavailable: 503,
} as const;
