import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { usePaginationItemLabel } from '../../../i18n/usePaginationItemLabel';
import { PageNavigation } from '../../../ui/components/PageNavigation';
import type { Order } from '../domain/order';
import { MyOrderList } from './MyOrderList';

type MyOrdersResultsProps = {
  readonly orders: readonly Order[];
  readonly now: Date;
  readonly page: number;
  readonly pageCount: number;
  readonly isRefreshing: boolean;
  readonly onPageChange: (page: number) => void;
};

export function MyOrdersResults({
  orders,
  now,
  page,
  pageCount,
  isRefreshing,
  onPageChange,
}: MyOrdersResultsProps) {
  const { t } = useTranslation();
  const getItemAriaLabel = usePaginationItemLabel();
  return (
    <Stack spacing={{ xs: 4, md: 5 }}>
      <Stack
        aria-busy={isRefreshing}
        sx={{
          opacity: isRefreshing ? 0.55 : 1,
          transition: (theme) => theme.transitions.create('opacity'),
        }}
      >
        <MyOrderList orders={orders} now={now} />
      </Stack>
      {pageCount > 1 && (
        <PageNavigation
          label={t('orders.list.paginationLabel')}
          page={page}
          pageCount={pageCount}
          onPageChange={onPageChange}
          getItemAriaLabel={getItemAriaLabel}
        />
      )}
    </Stack>
  );
}
