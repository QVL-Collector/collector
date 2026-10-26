import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { Outlet } from 'react-router';
import type { CollectorRole } from '../../auth/collectorRoles';
import { ErrorState } from '../../ui/components/ErrorState';
import { PendingState } from '../../ui/components/PendingState';
import { AccessDeniedPage } from '../pages/AccessDeniedPage';
import { useAuthenticationGate } from './useAuthenticationGate';

type RequireAuthenticationProps = {
  readonly requiredRole?: CollectorRole;
};

export function RequireAuthentication({ requiredRole }: RequireAuthenticationProps) {
  const { t } = useTranslation();
  const { status, retry } = useAuthenticationGate(requiredRole);
  switch (status) {
    case 'granted':
      return <Outlet />;
    case 'forbidden':
      return <AccessDeniedPage />;
    case 'redirecting':
      return <PendingState message={t('auth.gate.redirecting')} />;
    case 'unavailable':
      return (
        <ErrorState
          title={t('auth.gate.unavailableTitle')}
          description={t('auth.gate.unavailableDescription')}
          action={
            <Button variant="contained" onClick={retry}>
              {t('auth.gate.retry')}
            </Button>
          }
        />
      );
  }
}
