import { parseDate, parseUuid } from '../../../shared/marketplacePayload';
import {
  expectArray,
  expectInteger,
  expectNonEmptyString,
  expectRecord,
  InvalidPayloadError,
} from '../../../shared/payload';
import type { ArticlePhoto } from '../domain/articlePhoto';

const photoUrlProtocols: ReadonlySet<string> = new Set(['https:', 'http:']);

function protocolOf(text: string): string | null {
  try {
    return new URL(text).protocol;
  } catch {
    return null;
  }
}

function parsePhotoUrl(value: unknown, path: string): string {
  const text = expectNonEmptyString(value, path);
  const protocol = protocolOf(text);
  if (protocol === null || !photoUrlProtocols.has(protocol)) {
    throw new InvalidPayloadError(path);
  }
  return text;
}

export function parsePhoto(payload: unknown, path = 'photo'): ArticlePhoto {
  const photo = expectRecord(payload, path);
  return {
    id: parseUuid(photo.id, `${path}.id`),
    url: parsePhotoUrl(photo.url, `${path}.url`),
    expiresAt: parseDate(photo.expireLe, `${path}.expireLe`),
    width: expectInteger(photo.largeur, `${path}.largeur`, 1),
    height: expectInteger(photo.hauteur, `${path}.hauteur`, 1),
  };
}

export function parsePhotoList(payload: unknown): readonly ArticlePhoto[] {
  const root = expectRecord(payload, 'photos');
  return expectArray(root.elements, 'elements').map((item, index) =>
    parsePhoto(item, `elements[${String(index)}]`),
  );
}
