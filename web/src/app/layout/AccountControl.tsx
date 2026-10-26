import LoginIcon from '@mui/icons-material/Login';
import { ErrorToast } from '../../ui/components/ErrorToast';
import { ResponsiveActionButton } from '../../ui/components/ResponsiveActionButton';
import { SignedInAccount } from './SignedInAccount';
import { useAccountControl } from './useAccountControl';

export function AccountControl() {
  const model = useAccountControl();
  return (
    <>
      {model.identity === null ? (
        <ResponsiveActionButton
          label={model.labels.signIn}
          icon={<LoginIcon />}
          compact={model.compact}
          loading={model.busy}
          variant="contained"
          onClick={model.signIn}
        />
      ) : (
        <SignedInAccount
          identity={model.identity}
          labels={model.labels}
          canOpenSellerSpace={model.canOpenSellerSpace}
          compact={model.compact}
          busy={model.busy}
          onSignOut={model.signOut}
        />
      )}
      <ErrorToast
        open={model.failureVisible}
        message={model.failureMessage}
        onClose={model.dismissFailure}
      />
    </>
  );
}
