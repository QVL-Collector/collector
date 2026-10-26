import { parseOidcConfig, type OidcConfig } from '../auth/oidcConfig';
import { expectOneOf, expectRecord, expectString, InvalidPayloadError } from '../shared/payload';

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
  readonly catalogue: CatalogueConfig;
  readonly oidc: OidcConfig;
};

const sameOriginPathPattern = /^(\/[A-Za-z0-9._~-]+)*$/;

function parseSameOriginPath(value: unknown, path: string): string {
  const url = expectString(value, path);
  if (!sameOriginPathPattern.test(url)) {
    throw new InvalidPayloadError(path);
  }
  return url;
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
    catalogue: parseCatalogueConfig(root.catalogue),
    oidc: parseOidcConfig(root.oidc),
  };
}
