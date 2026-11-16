import { useAuthState } from '../../../auth/useAuthState';

export type ReservationAccess = 'reserved' | 'sign-in' | 'buyer-only' | 'available';

export function useReservationAccess(reserved: boolean): ReservationAccess {
  const state = useAuthState();
  if (reserved) {
    return 'reserved';
  }
  if (state.status === 'anonymous') {
    return 'sign-in';
  }
  return state.user.roles.includes('acheteur') ? 'available' : 'buyer-only';
}
