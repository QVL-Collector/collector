import EventBusyRounded from '@mui/icons-material/EventBusyRounded';
import EventRounded from '@mui/icons-material/EventRounded';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { ReactElement } from 'react';
import type { OrderStatus } from '../domain/order';
import type { OrderSummary } from '../hooks/useOrderSummary';

type DeadlineAppearance = {
  readonly icon: ReactElement;
  readonly iconColor: string;
  readonly dateColor: string;
};

const deadlineAppearances: Readonly<Record<OrderStatus, DeadlineAppearance>> = {
  en_attente_paiement: {
    icon: <EventRounded />,
    iconColor: 'secondary.main',
    dateColor: 'secondary.dark',
  },
  expiree: {
    icon: <EventBusyRounded />,
    iconColor: 'text.secondary',
    dateColor: 'text.secondary',
  },
};

type OrderDeadlineProps = {
  readonly summary: OrderSummary;
};

export function OrderDeadline({ summary }: OrderDeadlineProps) {
  const { icon, iconColor, dateColor } = deadlineAppearances[summary.status];
  return (
    <Stack direction="row" spacing={1.5} alignItems="flex-start">
      <Stack component="span" aria-hidden="true" color={iconColor} paddingTop={0.5}>
        {icon}
      </Stack>
      <Stack spacing={0.5}>
        <Typography variant="body2" color="text.secondary">
          {summary.deadlineTitle}
        </Typography>
        <Typography
          component="time"
          dateTime={summary.paymentDueDateTime}
          variant="subtitle1"
          color={dateColor}
        >
          {summary.paymentDueLabel}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {summary.deadlineHint}
        </Typography>
      </Stack>
    </Stack>
  );
}
