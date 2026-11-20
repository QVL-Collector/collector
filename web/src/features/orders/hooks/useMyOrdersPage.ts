import { useCallback, useState, type RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { useReconnect } from '../../../auth/useReconnect';
import { usePageIdentity } from '../../../navigation/usePageIdentity';
import { useNow } from '../../../ui/hooks/useNow';
import { toMyOrdersView, type MyOrdersView } from './myOrdersView';
import { useExpiryAnnouncement } from './useExpiryAnnouncement';
import { useMyOrders } from './useMyOrders';
import { useMyOrdersAnnouncement } from './useMyOrdersAnnouncement';
import { useRefetchOnDeadline } from './useRefetchOnDeadline';

const clockTickMs = 10_000;

export type MyOrdersPageModel = {
  readonly headingRef: RefObject<HTMLHeadingElement | null>;
  readonly view: MyOrdersView;
  readonly now: Date;
  readonly listAnnouncement: string;
  readonly expiryAnnouncement: string;
  readonly sessionExpiredDescription: string;
  readonly retry: () => void;
  readonly reconnect: () => void;
  readonly goToPage: (page: number) => void;
};

export function useMyOrdersPage(): MyOrdersPageModel {
  const { t } = useTranslation();
  const headingRef = usePageIdentity(t('orders.list.documentTitle'));
  const [page, setPage] = useState(1);
  const query = useMyOrders(page);
  const view = toMyOrdersView(query, page);
  const hasPendingOrder =
    query.data?.orders.some((order) => order.status === 'en_attente_paiement') ?? false;
  const now = useNow(clockTickMs, hasPendingOrder);
  const listAnnouncement = useMyOrdersAnnouncement(view);
  const expiryAnnouncement = useExpiryAnnouncement(query.data?.orders);
  const { reconnect, reconnectFailed } = useReconnect();
  const { refetch } = query;
  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);
  useRefetchOnDeadline(query.data?.orders, now, retry);
  return {
    headingRef,
    view,
    now,
    listAnnouncement,
    expiryAnnouncement,
    sessionExpiredDescription: reconnectFailed
      ? t('auth.signInFailed')
      : t('orders.list.unauthenticated.description'),
    retry,
    reconnect,
    goToPage: setPage,
  };
}
