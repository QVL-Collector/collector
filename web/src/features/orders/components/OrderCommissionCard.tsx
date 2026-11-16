import Card from '@mui/material/Card';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';
import { AmountRow } from '../../../ui/components/AmountRow';
import { FactList } from '../../../ui/components/FactList';
import type { OrderSummary } from '../hooks/useOrderSummary';

type OrderCommissionCardProps = {
  readonly summary: OrderSummary;
};

export function OrderCommissionCard({ summary }: OrderCommissionCardProps) {
  const { t } = useTranslation();
  const headingId = useId();
  return (
    <Card component="section" variant="outlined" aria-labelledby={headingId} sx={revealSx(2)}>
      <Stack spacing={2.5} padding={{ xs: 2.5, md: 3 }}>
        <Typography id={headingId} component="h2" variant="overline" color="text.secondary">
          {t('orders.splitTitle')}
        </Typography>
        <FactList>
          <AmountRow label={t('orders.commission')} amount={summary.commissionLabel} />
          <AmountRow label={t('orders.sellerAmount')} amount={summary.sellerAmountLabel} />
        </FactList>
        <Typography variant="body2" color="text.secondary">
          {t('orders.commissionHint')}
        </Typography>
      </Stack>
    </Card>
  );
}
