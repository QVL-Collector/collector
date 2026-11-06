export const languesE2e = ['fr', 'en'] as const;

export type Langue = (typeof languesE2e)[number];

const langueParDefaut: Langue = 'fr';

function estLangue(valeur: string): valeur is Langue {
  return languesE2e.some((langue) => langue === valeur);
}

export function lireLangues(valeur: string | undefined): readonly Langue[] {
  const demandees = (valeur ?? '')
    .split(',')
    .map((langue) => langue.trim().toLowerCase())
    .filter((langue) => langue !== '');
  if (demandees.length === 0) {
    return [langueParDefaut];
  }
  const inconnues = demandees.filter((langue) => !estLangue(langue));
  if (inconnues.length > 0) {
    throw new Error(
      `E2E_LANGUES : langue(s) inconnue(s) ${inconnues.join(', ')} (attendu : ${languesE2e.join(', ')})`,
    );
  }
  return [...new Set(demandees.filter(estLangue))];
}

export const localeNavigateur: Readonly<Record<Langue, string>> = {
  fr: 'fr-FR',
  en: 'en-GB',
};
