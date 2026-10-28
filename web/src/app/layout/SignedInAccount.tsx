import LogoutIcon from '@mui/icons-material/Logout';
import Stack from '@mui/material/Stack';
import type { UserIdentity } from '../../auth/useUserIdentity';
import { AccountBadge } from '../../ui/components/AccountBadge';
import { ResponsiveActionButton } from '../../ui/components/ResponsiveActionButton';
import { AdminSpaceLink } from './AdminSpaceLink';
import { SellerSpaceLink } from './SellerSpaceLink';
import type { AccountControlLabels } from './useAccountControl';

type SignedInAccountProps = {
  readonly identity: UserIdentity;
  readonly labels: AccountControlLabels;
  readonly canOpenSellerSpace: boolean;
  readonly canOpenAdminSpace: boolean;
  readonly compact: boolean;
  readonly busy: boolean;
  readonly onSignOut: () => void;
};

export function SignedInAccount({
  identity,
  labels,
  canOpenSellerSpace,
  canOpenAdminSpace,
  compact,
  busy,
  onSignOut,
}: SignedInAccountProps) {
  return (
    <Stack
      role="group"
      aria-label={labels.account}
      direction="row"
      alignItems="center"
      spacing={{ xs: 0.5, sm: 1.5 }}
      minWidth={0}
    >
      {canOpenAdminSpace && <AdminSpaceLink label={labels.adminSpace} compact={compact} />}
      {canOpenSellerSpace && <SellerSpaceLink label={labels.sellerSpace} compact={compact} />}
      <AccountBadge
        name={identity.name}
        detail={identity.roleLabel}
        initials={identity.initials}
        compact={compact}
      />
      <ResponsiveActionButton
        label={labels.signOut}
        icon={<LogoutIcon />}
        compact={compact}
        loading={busy}
        variant="outlined"
        onClick={onSignOut}
      />
    </Stack>
  );
}
