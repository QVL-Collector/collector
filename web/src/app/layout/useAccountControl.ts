import { useTranslation } from 'react-i18next';
import { useUserIdentity, type UserIdentity } from '../../auth/useUserIdentity';
import { useIsCompactScreen } from '../../ui/hooks/useIsCompactScreen';
import { useAuthActions, type AuthAction } from './useAuthActions';

export type AccountControlLabels = {
  readonly signIn: string;
  readonly signOut: string;
  readonly account: string;
  readonly sellerSpace: string;
};

export type AccountControlModel = {
  readonly identity: UserIdentity | null;
  readonly canOpenSellerSpace: boolean;
  readonly compact: boolean;
  readonly busy: boolean;
  readonly labels: AccountControlLabels;
  readonly failureMessage: string;
  readonly failureVisible: boolean;
  readonly signIn: () => void;
  readonly signOut: () => void;
  readonly dismissFailure: () => void;
};

const failureMessageKeys = {
  signIn: 'auth.signInFailed',
  signOut: 'auth.signOutFailed',
} as const satisfies Record<AuthAction, string>;

export function useAccountControl(): AccountControlModel {
  const { t } = useTranslation();
  const identity = useUserIdentity();
  const compact = useIsCompactScreen();
  const { busy, failedAction, failureVisible, signIn, signOut, dismissFailure } = useAuthActions();
  return {
    identity,
    canOpenSellerSpace: identity?.roles.includes('vendeur') ?? false,
    compact,
    busy,
    labels: {
      signIn: t('auth.signIn'),
      signOut: t('auth.signOut'),
      account: t('auth.accountLabel'),
      sellerSpace: t('seller.navigation'),
    },
    failureMessage: failedAction === null ? '' : t(failureMessageKeys[failedAction]),
    failureVisible,
    signIn,
    signOut,
    dismissFailure,
  };
}
