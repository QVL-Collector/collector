import ArrowBackRounded from '@mui/icons-material/ArrowBackRounded';
import VisibilityOutlined from '@mui/icons-material/VisibilityOutlined';
import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { LinkButton } from '../../../ui/components/LinkButton';
import { cataloguePath } from '../../catalogue/cataloguePaths';
import type { Order, OrderStatus } from '../domain/order';
import { useOrderSummary } from '../hooks/useOrderSummary';
import { OrderAmountsCard } from './OrderAmountsCard';
import { OrderCommissionCard } from './OrderCommissionCard';
import { OrderStatusPanel } from './OrderStatusPanel';

const articleLinkVariants: Readonly<Record<OrderStatus, 'contained' | 'outlined'>> = {
  en_attente_paiement: 'outlined',
  expiree: 'contained',
};

type OrderSummaryViewProps = {
  readonly order: Order;
};

export function OrderSummaryView({ order }: OrderSummaryViewProps) {
  const { t } = useTranslation();
  const summary = useOrderSummary(order);
  return (
    <Stack spacing={{ xs: 3, md: 4 }}>
      <OrderStatusPanel summary={summary} />
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={{ xs: 3, md: 4 }}>
        <Stack flex={1}>
          <OrderAmountsCard summary={summary} />
        </Stack>
        <Stack flex={1}>
          <OrderCommissionCard summary={summary} />
        </Stack>
      </Stack>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} alignItems={{ sm: 'center' }}>
        <LinkButton
          to={summary.articlePath}
          label={t('orders.viewArticle')}
          icon={<VisibilityOutlined />}
          variant={articleLinkVariants[summary.status]}
        />
        <LinkButton
          to={cataloguePath}
          label={t('common.backToCatalogue')}
          icon={<ArrowBackRounded />}
          variant="text"
        />
      </Stack>
    </Stack>
  );
}
