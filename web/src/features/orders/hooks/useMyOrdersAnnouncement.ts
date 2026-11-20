import { useTranslation } from 'react-i18next';
import type { MyOrdersView } from './myOrdersView';

export function useMyOrdersAnnouncement(view: MyOrdersView): string {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
    case 'unauthenticated':
    case 'forbidden':
    case 'error':
      return '';
    case 'empty':
      return t('orders.list.empty.title');
    case 'ready':
      return view.isRefreshing
        ? t('orders.list.loading')
        : t('orders.list.loaded', { count: view.totalItems });
  }
}
