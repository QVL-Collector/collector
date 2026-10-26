import type { User, UserProfile } from 'oidc-client-ts';
import type { AuthUser } from './AuthSession';
import { readRealmRoles } from './accessTokenClaims';

function displayNameOf(profile: UserProfile): string | null {
  const candidates = [profile.name, profile.preferred_username];
  return candidates.find((candidate) => candidate !== undefined && candidate.trim() !== '') ?? null;
}

export function toAuthUser(user: User): AuthUser {
  return { displayName: displayNameOf(user.profile), roles: readRealmRoles(user.access_token) };
}
