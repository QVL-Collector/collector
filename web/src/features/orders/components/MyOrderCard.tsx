import ChevronRightRounded from '@mui/icons-material/ChevronRightRounded';
import Card from '@mui/material/Card';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { Link as RouterLink } from 'react-router';
import { liftOnHoverSx } from '../../../ui/animations/liftOnHoverSx';
import { FactItem } from '../../../ui/components/FactItem';
import type { Order } from '../domain/order';
import { useMyOrderCard } from '../hooks/useMyOrderCard';
import { MyOrderDeadline } from './MyOrderDeadline';
import { OrderStatusChip } from './OrderStatusChip';

type MyOrderCardProps = {
  readonly order: Order;
  readonly now: Date;
};

export function MyOrderCard({ order, now }: MyOrderCardProps) {
  const card = useMyOrderCard(order, now);
  const titleId = useId();
  return (
    <Card component="article" variant="outlined" aria-labelledby={titleId} sx={liftOnHoverSx}>
      <Stack spacing={2} padding={{ xs: 2, sm: 2.5 }}>
        <Stack
          direction={{ xs: 'column', sm: 'row' }}
          spacing={1}
          justifyContent="space-between"
          alignItems={{ xs: 'flex-start', sm: 'center' }}
        >
          <OrderStatusChip status={card.status} label={card.statusLabel} />
          <Typography
            component="time"
            dateTime={card.createdDateTime}
            variant="body2"
            color="text.secondary"
          >
            {card.createdLabel}
          </Typography>
        </Stack>
        <Typography id={titleId} component="h3" variant="h6" sx={{ overflowWrap: 'anywhere' }}>
          <Link component={RouterLink} to={card.articlePath} color="inherit" underline="hover">
            {card.articleTitle}
          </Link>
        </Typography>
        <Stack
          component="dl"
          direction={{ xs: 'column', sm: 'row' }}
          spacing={{ xs: 2, sm: 5 }}
          margin={0}
        >
          <FactItem label={card.totalTitle}>{card.totalPaidLabel}</FactItem>
          <FactItem label={card.deadlineTitle}>
            <MyOrderDeadline
              status={card.status}
              paymentDueLabel={card.paymentDueLabel}
              paymentDueDateTime={card.paymentDueDateTime}
              timeLeftLabel={card.timeLeftLabel}
            />
          </FactItem>
        </Stack>
        <Stack direction="row" justifyContent="flex-end">
          <Link
            component={RouterLink}
            to={card.detailPath}
            aria-label={card.detailAccessibleLabel}
            variant="body2"
            underline="hover"
            fontWeight={600}
            display="inline-flex"
            alignItems="center"
            minHeight="2.75rem"
          >
            {card.detailLabel}
            <ChevronRightRounded fontSize="small" />
          </Link>
        </Stack>
      </Stack>
    </Card>
  );
}
