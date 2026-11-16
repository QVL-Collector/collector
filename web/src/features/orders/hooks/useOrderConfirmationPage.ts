import { useCallback, type RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router';
import { useReconnect } from '../../../auth/useReconnect';
import { usePageIdentity } from '../../../navigation/usePageIdentity';
import { toOrderConfirmationView, type OrderConfirmationView } from './orderConfirmationView';
import { useOrder } from './useOrder';
import { useOrderConfirmationText } from './useOrderConfirmationText';

export type OrderConfirmationPageModel = {
  readonly headingRef: RefObject<HTMLHeadingElement | null>;
  readonly title: string;
  readonly subtitle: string | undefined;
  readonly announcement: string;
  readonly view: OrderConfirmationView;
  readonly sessionExpiredDescription: string;
  readonly retry: () => void;
  readonly reconnect: () => void;
};

export function useOrderConfirmationPage(): OrderConfirmationPageModel {
  const { t } = useTranslation();
  const { orderId = '' } = useParams();
  const query = useOrder(orderId);
  const view = toOrderConfirmationView(query);
  const { documentTitle, ...text } = useOrderConfirmationText(view);
  const headingRef = usePageIdentity(documentTitle);
  const { reconnect, reconnectFailed } = useReconnect();
  const { refetch } = query;
  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);
  return {
    headingRef,
    ...text,
    view,
    sessionExpiredDescription: reconnectFailed
      ? t('auth.signInFailed')
      : t('orders.unauthenticated.description'),
    retry,
    reconnect,
  };
}
