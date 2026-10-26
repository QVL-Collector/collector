import { screen } from '@testing-library/react';
import axe from 'axe-core';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import {
  catalogueContaining,
  catalogueFailing,
  catalogueNeverAnswering,
  publishedArticle,
  publishedArticles,
  renderCatalogue,
  switchLanguage,
  type FakeCatalogue,
} from '../US-010/catalogueHarness';

const blockingImpacts = new Set(['serious', 'critical']);

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

type CatalogueState = readonly [
  state: string,
  language: SupportedLanguage,
  catalogue: () => FakeCatalogue,
  url: string,
  landmark: string,
];

const states: readonly CatalogueState[] = [
  ['liste', 'fr', () => catalogueContaining(publishedArticles(30)), '/', publishedArticle(1).title],
  ['liste', 'en', () => catalogueContaining(publishedArticles(30)), '/', publishedArticle(1).title],
  ['vide', 'fr', () => catalogueContaining([]), '/', 'Aucun article publié pour le moment'],
  ['vide', 'en', () => catalogueContaining([]), '/', 'No published items yet'],
  ['erreur', 'fr', catalogueFailing, '/', 'Le catalogue est momentanément indisponible'],
  ['erreur', 'en', catalogueFailing, '/', 'The catalogue is temporarily unavailable'],
  [
    'hors limites',
    'fr',
    () => catalogueContaining(publishedArticles(30)),
    '/?page=99',
    'Cette page du catalogue est vide',
  ],
  [
    'hors limites',
    'en',
    () => catalogueContaining(publishedArticles(30)),
    '/?page=99',
    'This catalogue page is empty',
  ],
  ['chargement', 'fr', catalogueNeverAnswering, '/', 'Chargement du catalogue en cours'],
  ['chargement', 'en', catalogueNeverAnswering, '/', 'Loading the catalogue'],
];

describe('US-025-AC4 axe-core sur les états du catalogue, en français et en anglais', () => {
  it.each(states)(
    'Given l’état %s en %s, When axe-core analyse la page, Then aucune violation serious ou critical',
    async (_, language, catalogue, url, landmark) => {
      await switchLanguage(language);
      renderCatalogue(catalogue(), url);
      await screen.findAllByText(landmark, {}, { timeout: 15000 });

      expect(document.documentElement).toHaveAttribute('lang', language);
      expect(await blockingViolations()).toEqual([]);
    },
    30000,
  );
});
