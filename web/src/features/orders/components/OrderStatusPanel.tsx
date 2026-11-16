import Card from '@mui/material/Card';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';
import type { OrderSummary } from '../hooks/useOrderSummary';
import { OrderDeadline } from './OrderDeadline';
import { OrderStatusChip } from './OrderStatusChip';

type OrderStatusPanelProps = {
  readonly summary: OrderSummary;
};

export function OrderStatusPanel({ summary }: OrderStatusPanelProps) {
  const { t } = useTranslation();
  const headingId = useId();
  return (
    <Card component="section" variant="outlined" aria-labelledby={headingId} sx={revealSx(0)}>
      <Stack
        direction={{ xs: 'column', sm: 'row' }}
        spacing={{ xs: 2.5, sm: 4 }}
        justifyContent="space-between"
        alignItems={{ xs: 'flex-start', sm: 'center' }}
        padding={{ xs: 2.5, md: 3 }}
      >
        <Stack spacing={1.5} alignItems="flex-start">
          <Typography id={headingId} component="h2" variant="overline" color="text.secondary">
            {t('orders.statusTitle')}
          </Typography>
          <OrderStatusChip status={summary.status} label={summary.statusLabel} />
          <Typography variant="caption" color="text.secondary">
            <time dateTime={summary.createdDateTime}>{summary.createdLabel}</time>
          </Typography>
        </Stack>
        <OrderDeadline summary={summary} />
      </Stack>
    </Card>
  );
}
