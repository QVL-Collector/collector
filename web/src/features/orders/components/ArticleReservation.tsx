import Stack from '@mui/material/Stack';
import { useArticleReservation } from '../hooks/useArticleReservation';
import { ReservationAccessContent } from './ReservationAccessContent';
import { ReservationFailureAlert } from './ReservationFailureAlert';

type ArticleReservationProps = {
  readonly articleId: string;
  readonly reserved: boolean;
};

export function ArticleReservation({ articleId, reserved }: ArticleReservationProps) {
  const reservation = useArticleReservation(reserved);
  return (
    <Stack spacing={1.5}>
      <ReservationAccessContent
        articleId={articleId}
        access={reservation.access}
        blocked={reservation.blocked}
        signInFailed={reservation.signInFailed}
        onStart={reservation.clearFailure}
        onFailure={reservation.reportFailure}
        onSignIn={reservation.signIn}
      />
      {reservation.failure !== null && (
        <ReservationFailureAlert
          failure={reservation.failure}
          alertRef={reservation.failureRef}
          onSignIn={reservation.signIn}
        />
      )}
    </Stack>
  );
}
