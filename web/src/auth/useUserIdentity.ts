import { useTranslation } from 'react-i18next';
import { initialsOf } from '../shared/initials';
import { primaryRole, type CollectorRole } from './collectorRoles';
import { useAuthState } from './useAuthState';

export type UserIdentity = {
  readonly name: string;
  readonly roleLabel: string;
  readonly initials: string;
  readonly roles: readonly CollectorRole[];
};

const roleLabelKeys = {
  admin: 'auth.roles.admin',
  vendeur: 'auth.roles.vendeur',
  acheteur: 'auth.roles.acheteur',
} as const satisfies Record<CollectorRole, string>;

export function useUserIdentity(): UserIdentity | null {
  const { t } = useTranslation();
  const state = useAuthState();
  if (state.status === 'anonymous') {
    return null;
  }
  const name = state.user.displayName ?? t('auth.unknownUser');
  const role = primaryRole(state.user.roles);
  return {
    name,
    roleLabel: role === null ? t('auth.roles.none') : t(roleLabelKeys[role]),
    initials: initialsOf(name),
    roles: state.user.roles,
  };
}
