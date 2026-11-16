import LoginRounded from '@mui/icons-material/LoginRounded';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import type { RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';
import type { ReservationFailure } from '../hooks/reservationFailure';
import { ReservationFailureMessage } from './ReservationFailureMessage';

type ReservationFailureAlertProps = {
  readonly failure: ReservationFailure;
  readonly alertRef: RefObject<HTMLDivElement | null>;
  readonly onSignIn: () => void;
};

export function ReservationFailureAlert({
  failure,
  alertRef,
  onSignIn,
}: ReservationFailureAlertProps) {
  const { t } = useTranslation();
  return (
    <Alert
      ref={alertRef}
      tabIndex={-1}
      severity="error"
      action={
        failure === 'session-expired' ? (
          <Button
            color="inherit"
            size="small"
            startIcon={<LoginRounded />}
            onClick={onSignIn}
            sx={{ minHeight: '2.75rem' }}
          >
            {t('article.reservation.reconnect')}
          </Button>
        ) : undefined
      }
      sx={revealSx(0)}
    >
      <ReservationFailureMessage failure={failure} />
    </Alert>
  );
}
