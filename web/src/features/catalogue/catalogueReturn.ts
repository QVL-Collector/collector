import { isRecord } from '../../shared/payload';
import { cataloguePath } from './cataloguePaths';

export type CatalogueReturnState = {
  readonly catalogueSearch: string;
};

const searchPattern = /^\?[^#]*$/;

export function toCatalogueReturnState(search: string): CatalogueReturnState {
  return { catalogueSearch: search };
}

export function catalogueReturnPath(state: unknown): string {
  if (!isRecord(state) || typeof state.catalogueSearch !== 'string') {
    return cataloguePath;
  }
  const search = state.catalogueSearch;
  return searchPattern.test(search) ? `${cataloguePath}${search}` : cataloguePath;
}
