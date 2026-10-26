export const supportedLanguages = ['fr', 'en'] as const;

export type SupportedLanguage = (typeof supportedLanguages)[number];

export const defaultLanguage: SupportedLanguage = 'fr';

export const localeTags: Readonly<Record<SupportedLanguage, string>> = {
  fr: 'fr-FR',
  en: 'en-GB',
};

export function toSupportedLanguage(value: string | undefined): SupportedLanguage | undefined {
  const primarySubtag = value?.toLowerCase().split('-')[0];
  return supportedLanguages.find((language) => language === primarySubtag);
}

export function detectInitialLanguage(): SupportedLanguage {
  const preferred = navigator.languages
    .map(toSupportedLanguage)
    .find((language) => language !== undefined);
  return preferred ?? defaultLanguage;
}
