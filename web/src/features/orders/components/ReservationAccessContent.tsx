import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';
import type { ReservationFailure } from '../hooks/reservationFailure';
import type { ReservationAccess } from '../hooks/useReservationAccess';
import { ReservationSignInPrompt } from './ReservationSignInPrompt';
import { ReservationUnavailable } from './ReservationUnavailable';
import { ReserveArticleButton } from './ReserveArticleButton';

type ReservationAccessContentProps = {
  readonly articleId: string;
  readonly access: ReservationAccess;
  readonly blocked: boolean;
  readonly signInFailed: boolean;
  readonly onStart: () => void;
  readonly onFailure: (failure: ReservationFailure) => void;
  readonly onSignIn: () => void;
};

export function ReservationAccessContent({
  articleId,
  access,
  blocked,
  signInFailed,
  onStart,
  onFailure,
  onSignIn,
}: ReservationAccessContentProps) {
  const { t } = useTranslation();
  switch (access) {
    case 'available':
      return (
        <ReserveArticleButton
          articleId={articleId}
          blocked={blocked}
          onStart={onStart}
          onFailure={onFailure}
        />
      );
    case 'reserved':
      return <ReservationUnavailable />;
    case 'sign-in':
      return <ReservationSignInPrompt signInFailed={signInFailed} onSignIn={onSignIn} />;
    case 'buyer-only':
      return (
        <Typography variant="body2" color="text.secondary">
          {t('article.reservation.buyerOnly')}
        </Typography>
      );
  }
}
