import Card from '@mui/material/Card';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';
import { AmountRow } from '../../../ui/components/AmountRow';
import { FactList } from '../../../ui/components/FactList';
import type { OrderSummary } from '../hooks/useOrderSummary';

type OrderAmountsCardProps = {
  readonly summary: OrderSummary;
};

export function OrderAmountsCard({ summary }: OrderAmountsCardProps) {
  const { t } = useTranslation();
  const headingId = useId();
  return (
    <Card component="section" variant="outlined" aria-labelledby={headingId} sx={revealSx(1)}>
      <Stack spacing={2.5} padding={{ xs: 2.5, md: 3 }}>
        <Typography id={headingId} component="h2" variant="overline" color="text.secondary">
          {t('orders.amountsTitle')}
        </Typography>
        <FactList>
          <AmountRow label={t('orders.price')} amount={summary.priceLabel} />
          <AmountRow label={t('orders.shippingCost')} amount={summary.shippingLabel} />
          <AmountRow label={t('orders.totalPaid')} amount={summary.totalPaidLabel} emphasized />
        </FactList>
      </Stack>
    </Card>
  );
}
