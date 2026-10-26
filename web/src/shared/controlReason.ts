export const controlReasons = [
  'annonce_incomplete',
  'coordonnees_detectees',
  'compte_non_autorise',
  'prix_a_confirmer',
  'vendeur_a_verifier',
  'categorie_inconnue',
] as const;

export type ControlReason = (typeof controlReasons)[number];
