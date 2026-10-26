import { execFile } from 'node:child_process';
import { copyFileSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { basename, join, resolve } from 'node:path';
import { afterAll, afterEach, beforeAll, describe, expect, it } from 'vitest';

const webRoot = resolve(import.meta.dirname, '../../..');
const paritySource = join(webRoot, 'scripts', 'check-translations.ts');
const eslintBin = join(webRoot, 'node_modules', 'eslint', 'bin', 'eslint.js');
const slowProcess = 60000;
const hardCodedTextRules = /^(i18next\/no-literal-string|no-restricted-syntax)$/;

type Outcome = { readonly status: number | null; readonly output: string };

type Catalogue = Record<string, unknown>;

const completeFrench: Catalogue = {
  app: { name: 'Collector', welcome: 'Bonjour {{name}}' },
  catalogue: { title: 'Catalogue', empty: { title: 'Aucun article' } },
};

const completeEnglish: Catalogue = {
  app: { name: 'Collector', welcome: 'Hello {{name}}' },
  catalogue: { title: 'Catalogue', empty: { title: 'No items' } },
};

const sandboxes: string[] = [];

afterEach(() => {
  for (const sandbox of sandboxes.splice(0)) {
    rmSync(sandbox, { recursive: true, force: true });
  }
});

function run(args: readonly string[], cwd: string): Promise<Outcome> {
  return new Promise((resolveOutcome) => {
    execFile(
      process.execPath,
      args,
      { cwd, maxBuffer: 16 * 1024 * 1024 },
      (error, stdout, stderr) => {
        resolveOutcome({
          status: error === null ? 0 : typeof error.code === 'number' ? error.code : 1,
          output: `${stdout}${stderr}`,
        });
      },
    );
  });
}

function locale(name: string, catalogue: Catalogue): string {
  return `export const ${name} = ${JSON.stringify(catalogue, null, 2)};\n`;
}

function runParityOn(french: Catalogue, english: Catalogue): Promise<Outcome> {
  const sandbox = mkdtempSync(join(tmpdir(), 'collector-parite-'));
  sandboxes.push(sandbox);
  const localesDirectory = join(sandbox, 'src', 'i18n', 'locales');
  mkdirSync(join(sandbox, 'scripts'), { recursive: true });
  mkdirSync(localesDirectory, { recursive: true });
  writeFileSync(join(sandbox, 'package.json'), '{ "type": "module" }\n');
  copyFileSync(paritySource, join(sandbox, 'scripts', 'check-translations.ts'));
  writeFileSync(join(localesDirectory, 'fr.ts'), locale('fr', french));
  writeFileSync(join(localesDirectory, 'en.ts'), locale('en', english));
  return run([join('scripts', 'check-translations.ts')], sandbox);
}

function withEnglish(patch: (english: Catalogue) => void): Catalogue {
  const english = structuredClone(completeEnglish);
  patch(english);
  return english;
}

function withFrench(patch: (french: Catalogue) => void): Catalogue {
  const french = structuredClone(completeFrench);
  patch(french);
  return french;
}

function section(catalogue: Catalogue, name: string): Catalogue {
  return catalogue[name] as Catalogue;
}

type LintMessage = { readonly ruleId: string | null; readonly fatal?: boolean };

type LintReport = { readonly filePath: string; readonly messages: readonly LintMessage[] };

type Probe = 'traduit' | 'texteJsx' | 'ariaLabel' | 'alt';

let lintReports: readonly LintReport[] = [];

function rulesReportedFor(probe: Probe): readonly LintMessage[] {
  const report = lintReports.find(
    (candidate) => basename(candidate.filePath) === `Sonde${probe}.tsx`,
  );
  if (report === undefined) {
    throw new Error(`Aucun rapport de lint pour ${probe}`);
  }
  return report.messages;
}

const translatedComponent = `import { useTranslation } from 'react-i18next';

export function SondeTexteEnDur() {
  const { t } = useTranslation();
  return <p aria-label={t('app.name')}>{t('app.name')}</p>;
}
`;

const literalTextComponent = `export function SondeTexteEnDur() {
  return <p>Bienvenue sur Collector</p>;
}
`;

const literalAccessibleNameComponent = `import { useTranslation } from 'react-i18next';

export function SondeTexteEnDur() {
  const { t } = useTranslation();
  return <button type="button" aria-label="Fermer la fenêtre">{t('app.name')}</button>;
}
`;

const literalAltComponent = `export function SondeTexteEnDur() {
  return <img src="/logo.svg" alt="Logo de Collector" />;
}
`;

describe('US-025-AC2 script de parité des traductions', () => {
  it(
    'Given les traductions réelles du site, When le script de parité s’exécute, Then il réussit',
    async () => {
      expect((await run([paritySource], webRoot)).status).toBe(0);
    },
    slowProcess,
  );

  it(
    'Given deux langues complètes et cohérentes, When le script s’exécute, Then il réussit',
    async () => {
      expect((await runParityOn(completeFrench, completeEnglish)).status).toBe(0);
    },
    slowProcess,
  );

  it(
    'Given une clé absente en anglais, When le script s’exécute, Then il échoue en nommant la clé',
    async () => {
      const english = withEnglish((catalogue) => {
        delete section(catalogue, 'catalogue').title;
      });

      const outcome = await runParityOn(completeFrench, english);

      expect(outcome.status).not.toBe(0);
      expect(outcome.output).toContain('catalogue.title');
    },
    slowProcess,
  );

  it(
    'Given une clé absente en français, When le script s’exécute, Then il échoue en nommant la clé',
    async () => {
      const french = withFrench((catalogue) => {
        delete section(section(catalogue, 'catalogue'), 'empty').title;
      });

      const outcome = await runParityOn(french, completeEnglish);

      expect(outcome.status).not.toBe(0);
      expect(outcome.output).toContain('catalogue.empty.title');
    },
    slowProcess,
  );

  it(
    'Given une section entière absente en anglais, When le script s’exécute, Then il échoue',
    async () => {
      const english = withEnglish((catalogue) => {
        delete catalogue.catalogue;
      });

      expect((await runParityOn(completeFrench, english)).status).not.toBe(0);
    },
    slowProcess,
  );

  it(
    'Given un texte anglais vide, When le script s’exécute, Then il échoue',
    async () => {
      const english = withEnglish((catalogue) => {
        section(catalogue, 'catalogue').title = '   ';
      });

      expect((await runParityOn(completeFrench, english)).status).not.toBe(0);
    },
    slowProcess,
  );

  it(
    'Given une variable différente entre les langues, When le script s’exécute, Then il échoue',
    async () => {
      const english = withEnglish((catalogue) => {
        section(catalogue, 'app').welcome = 'Hello {{username}}';
      });

      expect((await runParityOn(completeFrench, english)).status).not.toBe(0);
    },
    slowProcess,
  );
});

describe('US-025-AC2 lint contre le texte en dur', () => {
  const probeDirectory = join(webRoot, 'src', 'features', `sonde-us025-${String(process.pid)}`);
  const sources: Record<Probe, string> = {
    traduit: translatedComponent,
    texteJsx: literalTextComponent,
    ariaLabel: literalAccessibleNameComponent,
    alt: literalAltComponent,
  };

  beforeAll(async () => {
    mkdirSync(probeDirectory, { recursive: true });
    for (const [probe, source] of Object.entries(sources)) {
      writeFileSync(join(probeDirectory, `Sonde${probe}.tsx`), source);
    }
    const outcome = await run([eslintBin, '--format', 'json', probeDirectory], webRoot);
    lintReports = JSON.parse(outcome.output.slice(outcome.output.indexOf('['))) as LintReport[];
  }, slowProcess);

  afterAll(() => {
    rmSync(probeDirectory, { recursive: true, force: true });
  });

  it('Given un composant entièrement traduit, When le lint s’exécute, Then aucun problème n’est signalé', () => {
    expect(rulesReportedFor('traduit')).toEqual([]);
  });

  it.each([
    ['un texte JSX en dur', 'texteJsx'],
    ['un aria-label en dur', 'ariaLabel'],
    ['un alt en dur', 'alt'],
  ] as const)(
    'Given un composant avec %s, When le lint s’exécute, Then une règle le signale en erreur',
    (_, probe) => {
      const messages = rulesReportedFor(probe);

      expect(messages.some((message) => message.fatal === true)).toBe(false);
      expect(messages.map((message) => message.ruleId)).toEqual(
        expect.arrayContaining([expect.stringMatching(hardCodedTextRules)]),
      );
    },
  );
});
