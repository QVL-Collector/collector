import { useTranslation } from 'react-i18next';
import type { OrderConfirmationView } from './orderConfirmationView';

export type OrderConfirmationText = {
  readonly title: string;
  readonly subtitle: string | undefined;
  readonly documentTitle: string;
  readonly announcement: string;
};

export function useOrderConfirmationText(view: OrderConfirmationView): OrderConfirmationText {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
      return {
        title: t('orders.pendingTitle'),
        subtitle: undefined,
        documentTitle: t('orders.pendingDocumentTitle'),
        announcement: t('orders.loading'),
      };
    case 'unauthenticated':
    case 'forbidden':
    case 'error':
      return {
        title: t('orders.pendingTitle'),
        subtitle: undefined,
        documentTitle: t('orders.pendingDocumentTitle'),
        announcement: '',
      };
    case 'not-found':
      return {
        title: t('orders.notFound.title'),
        subtitle: t('orders.notFound.description'),
        documentTitle: t('orders.notFound.documentTitle'),
        announcement: t('orders.notFound.title'),
      };
    case 'ready': {
      const { status } = view.order;
      return {
        title: t(`orders.views.${status}.title`),
        subtitle: t(`orders.views.${status}.subtitle`),
        documentTitle: t(`orders.views.${status}.documentTitle`),
        announcement: t(`orders.views.${status}.announcement`),
      };
    }
  }
}
