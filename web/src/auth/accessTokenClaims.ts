import { isRecord } from '../shared/payload';
import { toCollectorRoles, type CollectorRole } from './collectorRoles';

function decodeBase64Url(segment: string): string {
  const base64 = segment.replace(/-/g, '+').replace(/_/g, '/');
  const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=');
  const bytes = Uint8Array.from(atob(padded), (character) => character.charCodeAt(0));
  return new TextDecoder().decode(bytes);
}

function readPayload(token: string): unknown {
  const segment = token.split('.')[1];
  if (segment === undefined) {
    return null;
  }
  try {
    const payload: unknown = JSON.parse(decodeBase64Url(segment));
    return payload;
  } catch {
    return null;
  }
}

export function readRealmRoles(accessToken: string): readonly CollectorRole[] {
  const payload = readPayload(accessToken);
  if (!isRecord(payload) || !isRecord(payload.realm_access)) {
    return [];
  }
  const roles: unknown = payload.realm_access.roles;
  return Array.isArray(roles) ? toCollectorRoles(roles) : [];
}
