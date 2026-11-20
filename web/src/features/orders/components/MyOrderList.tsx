import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';
import type { Order } from '../domain/order';
import { MyOrderCard } from './MyOrderCard';

type MyOrderListProps = {
  readonly orders: readonly Order[];
  readonly now: Date;
};

export function MyOrderList({ orders, now }: MyOrderListProps) {
  const { t } = useTranslation();
  return (
    <Stack
      component="ul"
      aria-label={t('orders.list.listLabel')}
      spacing={{ xs: 1.5, md: 2 }}
      margin={0}
      padding={0}
      maxWidth="48rem"
      sx={{ listStyle: 'none' }}
    >
      {orders.map((order, index) => (
        <Stack component="li" key={order.id} sx={revealSx(index)}>
          <MyOrderCard order={order} now={now} />
        </Stack>
      ))}
    </Stack>
  );
}
