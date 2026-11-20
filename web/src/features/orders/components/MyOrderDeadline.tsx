import Typography from '@mui/material/Typography';
import type { OrderStatus } from '../domain/order';

const dateColors: Readonly<Record<OrderStatus, string>> = {
  en_attente_paiement: 'secondary.dark',
  expiree: 'text.secondary',
};

type MyOrderDeadlineProps = {
  readonly status: OrderStatus;
  readonly paymentDueLabel: string;
  readonly paymentDueDateTime: string;
  readonly timeLeftLabel: string | null;
};

export function MyOrderDeadline({
  status,
  paymentDueLabel,
  paymentDueDateTime,
  timeLeftLabel,
}: MyOrderDeadlineProps) {
  return (
    <>
      <Typography
        component="time"
        dateTime={paymentDueDateTime}
        variant="inherit"
        color={dateColors[status]}
        display="block"
      >
        {paymentDueLabel}
      </Typography>
      {timeLeftLabel !== null && (
        <Typography component="span" variant="body2" color="text.secondary" display="block">
          {timeLeftLabel}
        </Typography>
      )}
    </>
  );
}
