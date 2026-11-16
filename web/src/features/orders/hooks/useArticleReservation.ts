import { useCallback, useEffect, useRef, useState, type RefObject } from 'react';
import { useReconnect } from '../../../auth/useReconnect';
import { blocksReservation, type ReservationFailure } from './reservationFailure';
import { useReservationAccess, type ReservationAccess } from './useReservationAccess';

export type ArticleReservationModel = {
  readonly access: ReservationAccess;
  readonly blocked: boolean;
  readonly failure: ReservationFailure | null;
  readonly failureRef: RefObject<HTMLDivElement | null>;
  readonly clearFailure: () => void;
  readonly reportFailure: (failure: ReservationFailure) => void;
  readonly signIn: () => void;
  readonly signInFailed: boolean;
};

export function useArticleReservation(reserved: boolean): ArticleReservationModel {
  const access = useReservationAccess(reserved);
  const { reconnect, reconnectFailed } = useReconnect();
  const [failure, setFailure] = useState<ReservationFailure | null>(null);
  const failureRef = useRef<HTMLDivElement>(null);
  const clearFailure = useCallback(() => {
    setFailure(null);
  }, []);
  useEffect(() => {
    if (failure !== null) {
      failureRef.current?.focus();
    }
  }, [failure]);
  return {
    access,
    blocked: blocksReservation(failure),
    failure,
    failureRef,
    clearFailure,
    reportFailure: setFailure,
    signIn: reconnect,
    signInFailed: reconnectFailed,
  };
}
