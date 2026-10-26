import { screen } from '@testing-library/react';
import axe from 'axe-core';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import {
  authorizationEndpoint,
  buyer,
  defaultConfig,
  openCollector,
  press,
  returnFromIdentityProvider,
  sellerA,
  serveConfig,
  signInAs,
  waitForNavigationTo,
} from '../US-008/identityProviderHarness';

const blockingImpacts = new Set(['serious', 'critical']);
const axeTimeout = 30000;

type Wording = {
  readonly signIn: string;
  readonly catalogue: string;
  readonly sellerSpace: string;
  readonly accessDenied: string;
  readonly signInFailed: string;
  readonly notFound: string;
  readonly bootstrapFailure: string;
};

const wording: Readonly<Record<SupportedLanguage, Wording>> = {
  fr: {
    signIn: 'Se connecter',
    catalogue: 'Catalogue',
    sellerSpace: 'Espace vendeur',
    accessDenied: 'Accès réservé',
    signInFailed: 'La connexion n’a pas abouti',
    notFound: 'Page introuvable',
    bootstrapFailure: 'L’application n’a pas pu démarrer',
  },
  en: {
    signIn: 'Sign in',
    catalogue: 'Catalogue',
    sellerSpace: 'Seller area',
    accessDenied: 'Restricted access',
    signInFailed: 'Sign-in did not complete',
    notFound: 'Page not found',
    bootstrapFailure: 'The application could not start',
  },
};

async function blockingViolations(): Promise<string[]> {
  const results = await axe.run(document, {
    resultTypes: ['violations'],
    rules: { 'color-contrast': { enabled: false } },
  });
  return results.violations
    .filter((violation) => blockingImpacts.has(violation.impact ?? ''))
    .map(
      (violation) =>
        `${violation.id} (${violation.impact ?? ''}) : ${violation.nodes.map((node) => node.target.join(' ')).join(', ')}`,
    );
}

async function expectDocumentLanguage(language: SupportedLanguage): Promise<void> {
  await expect.poll(() => document.documentElement.getAttribute('lang')).toBe(language);
}

const languages: readonly SupportedLanguage[] = ['fr', 'en'];

describe('US-025-AC4 axe-core sur les écrans existants, en français et en anglais', () => {
  it.each(languages)(
    'Given le catalogue en visiteur (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      await openCollector('/', language);
      await screen.findByRole('button', { name: wording[language].signIn });
      await screen.findByText('Objet de collection 1');
      await expectDocumentLanguage(language);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(languages)(
    'Given le catalogue du vendeur A connecté (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      await signInAs(sellerA, '/', language);
      await screen.findByText('Alice Vendeur');
      await screen.findByText('Objet de collection 1');
      await expectDocumentLanguage(language);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(languages)(
    'Given l’espace vendeur (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      await signInAs(sellerA, '/vendeur', language);
      await screen.findByRole('heading', { level: 1, name: wording[language].sellerSpace });
      await expectDocumentLanguage(language);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(languages)(
    'Given la page d’accès refusé (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      await signInAs(buyer, '/vendeur', language);
      await screen.findByRole('heading', { level: 1, name: wording[language].accessDenied });
      await expectDocumentLanguage(language);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(languages)(
    'Given la page d’échec de connexion (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      await openCollector('/', language);
      await press(await screen.findByRole('button', { name: wording[language].signIn }));
      await waitForNavigationTo(authorizationEndpoint());
      await returnFromIdentityProvider(
        (state) => `error=access_denied&state=${encodeURIComponent(state)}`,
      );
      await screen.findByText(wording[language].signInFailed);
      await expectDocumentLanguage(language);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(languages)(
    'Given la page introuvable (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      await openCollector('/adresse-inexistante', language);
      await screen.findByRole('heading', { level: 1, name: wording[language].notFound });
      await expectDocumentLanguage(language);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it.each(languages)(
    'Given l’échec de démarrage (%s), When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      serveConfig({ ...defaultConfig, oidc: { ...defaultConfig.oidc, scope: 'profile' } });
      await openCollector('/', language);
      await screen.findByText(wording[language].bootstrapFailure);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );
});
