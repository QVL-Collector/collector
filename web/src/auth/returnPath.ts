import { homePath, signInCallbackPath } from './authPaths';

export function toSafeReturnPath(value: unknown, origin: string): string {
  if (typeof value !== 'string' || !value.startsWith('/')) {
    return homePath;
  }
  const url = new URL(value, origin);
  if (
    url.origin !== origin ||
    url.pathname.startsWith('//') ||
    url.pathname.startsWith('/\\') ||
    url.pathname === signInCallbackPath
  ) {
    return homePath;
  }
  return `${url.pathname}${url.search}`;
}
