import { use } from 'react';
import type { AuthSession } from './AuthSession';
import { AuthSessionContext } from './AuthSessionContext';

export function useAuthSession(): AuthSession {
  return use(AuthSessionContext);
}
