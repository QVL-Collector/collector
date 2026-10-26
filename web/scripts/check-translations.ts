import { en } from '../src/i18n/locales/en.ts';
import { fr } from '../src/i18n/locales/fr.ts';

type TranslationNode = string | { readonly [key: string]: TranslationNode };

const placeholderPattern = /\{\{\s*([\w.]+)\s*\}\}/g;

const reference = { language: 'fr', translations: fr };
const candidates = [{ language: 'en', translations: en }];

function flatten(node: TranslationNode, prefix = ''): ReadonlyMap<string, string> {
  if (typeof node === 'string') {
    return new Map([[prefix, node]]);
  }
  return new Map(
    Object.entries(node).flatMap(([key, child]) => [
      ...flatten(child, prefix === '' ? key : `${prefix}.${key}`),
    ]),
  );
}

function placeholdersOf(text: string): string {
  return [...text.matchAll(placeholderPattern)]
    .map(([, name]) => name)
    .sort()
    .join(',');
}

function compare(
  language: string,
  expected: ReadonlyMap<string, string>,
  actual: ReadonlyMap<string, string>,
): string[] {
  const missing = [...expected.keys()]
    .filter((key) => !actual.has(key))
    .map((key) => `${language}: clé manquante ${key}`);
  const extra = [...actual.keys()]
    .filter((key) => !expected.has(key))
    .map((key) => `${language}: clé en trop ${key}`);
  const empty = [...actual]
    .filter(([, text]) => text.trim() === '')
    .map(([key]) => `${language}: texte vide ${key}`);
  const placeholders = [...actual]
    .filter(([key, text]) => {
      const source = expected.get(key);
      return source !== undefined && placeholdersOf(source) !== placeholdersOf(text);
    })
    .map(([key]) => `${language}: variables différentes ${key}`);
  return [...missing, ...extra, ...empty, ...placeholders];
}

const expected = flatten(reference.translations);
const errors = [
  ...compare(reference.language, expected, expected),
  ...candidates.flatMap(({ language, translations }) =>
    compare(language, expected, flatten(translations)),
  ),
];

if (errors.length > 0) {
  console.error(errors.join('\n'));
  process.exit(1);
}

const languages = [reference, ...candidates].map(({ language }) => language).join(', ');
console.log(`Traductions alignées : ${String(expected.size)} clés, langues ${languages}.`);
