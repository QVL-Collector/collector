import { useTranslation } from 'react-i18next';
import type { ReservationFailure } from '../hooks/reservationFailure';

type ReservationFailureMessageProps = {
  readonly failure: ReservationFailure;
};

export function ReservationFailureMessage({ failure }: ReservationFailureMessageProps) {
  const { t } = useTranslation();
  switch (failure) {
    case 'already-reserved':
      return t('article.reservation.failures.alreadyReserved');
    case 'reservation-already-active':
      return t('article.reservation.failures.reservationAlreadyActive');
    case 'rate-limited':
      return t('article.reservation.failures.rateLimited');
    case 'not-reservable':
      return t('article.reservation.failures.notReservable');
    case 'session-expired':
      return t('article.reservation.failures.sessionExpired');
    case 'forbidden':
      return t('article.reservation.failures.forbidden');
    case 'unavailable':
      return t('article.reservation.failures.unavailable');
  }
}
