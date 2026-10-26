import { createContext, type ReactNode } from 'react';
import { anonymousSession, type AuthSession } from './AuthSession';

export const AuthSessionContext = createContext<AuthSession>(anonymousSession);

type AuthSessionProviderProps = {
  readonly session: AuthSession;
  readonly children: ReactNode;
};

export function AuthSessionProvider({ session, children }: AuthSessionProviderProps) {
  return <AuthSessionContext value={session}>{children}</AuthSessionContext>;
}
