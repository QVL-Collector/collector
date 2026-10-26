import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { usePageIdentity } from '../../navigation/usePageIdentity';
import { PageHeading } from '../../ui/components/PageHeading';
import { PendingState } from '../../ui/components/PendingState';
import { BackToCatalogueButton } from './BackToCatalogueButton';
import { useSignInCallback } from './useSignInCallback';

export function SignInCallbackPage() {
  const { t } = useTranslation();
  const headingRef = usePageIdentity(t('auth.callback.documentTitle'));
  const status = useSignInCallback();
  if (status === 'pending') {
    return <PendingState message={t('auth.callback.pending')} />;
  }
  return (
    <Stack spacing={4} alignItems="flex-start">
      <PageHeading
        headingRef={headingRef}
        title={t('auth.callback.title')}
        subtitle={t('auth.callback.description')}
      />
      <BackToCatalogueButton />
    </Stack>
  );
}
