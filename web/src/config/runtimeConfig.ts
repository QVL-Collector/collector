import { parseOidcConfig, type OidcConfig } from '../auth/oidcConfig';
import type { PhotoLimits } from '../features/photos/domain/articlePhoto';
import { expectOneOf, expectRecord, expectString, InvalidPayloadError } from '../shared/payload';
import { parsePhotoLimits } from './photoLimitsConfig';

export const catalogueSources = ['api', 'demo'] as const;
export type CatalogueSource = (typeof catalogueSources)[number];

export const demoScenarios = ['nominal', 'empty', 'error'] as const;
export type DemoScenario = (typeof demoScenarios)[number];

export type CatalogueConfig = {
  readonly source: CatalogueSource;
  readonly demoScenario: DemoScenario;
};

export type RuntimeConfig = {
  readonly marketplaceApiUrl: string;
  readonly interactionsApiUrl: string;
  readonly transactionsApiUrl: string;
  readonly catalogue: CatalogueConfig;
  readonly oidc: OidcConfig;
  readonly photos?: PhotoLimits;
};

export const defaultInteractionsApiUrl = '/interactions';

export const defaultTransactionsApiUrl = '/transactions';

const sameOriginPathPattern = /^(\/[A-Za-z0-9._~-]+)*$/;

function parseSameOriginPath(value: unknown, path: string): string {
  const url = expectString(value, path);
  if (!sameOriginPathPattern.test(url)) {
    throw new InvalidPayloadError(path);
  }
  return url;
}

function parseOptionalSameOriginPath(value: unknown, path: string, fallback: string): string {
  return value === undefined ? fallback : parseSameOriginPath(value, path);
}

function parseCatalogueConfig(value: unknown): CatalogueConfig {
  const catalogue = expectRecord(value, 'catalogue');
  return {
    source: expectOneOf(catalogue.source, catalogueSources, 'catalogue.source'),
    demoScenario:
      catalogue.demoScenario === undefined
        ? 'nominal'
        : expectOneOf(catalogue.demoScenario, demoScenarios, 'catalogue.demoScenario'),
  };
}

export function parseRuntimeConfig(value: unknown): RuntimeConfig {
  const root = expectRecord(value, 'config');
  return {
    marketplaceApiUrl: parseSameOriginPath(root.marketplaceApiUrl, 'marketplaceApiUrl'),
    interactionsApiUrl: parseOptionalSameOriginPath(
      root.interactionsApiUrl,
      'interactionsApiUrl',
      defaultInteractionsApiUrl,
    ),
    transactionsApiUrl: parseOptionalSameOriginPath(
      root.transactionsApiUrl,
      'transactionsApiUrl',
      defaultTransactionsApiUrl,
    ),
    catalogue: parseCatalogueConfig(root.catalogue),
    oidc: parseOidcConfig(root.oidc),
    photos: parsePhotoLimits(root.photos),
  };
}
