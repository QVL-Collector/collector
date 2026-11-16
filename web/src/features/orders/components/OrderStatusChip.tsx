import HourglassTopRounded from '@mui/icons-material/HourglassTopRounded';
import Chip, { type ChipProps } from '@mui/material/Chip';
import type { ReactElement } from 'react';
import type { OrderStatus } from '../domain/order';

type StatusAppearance = {
  readonly icon: ReactElement;
  readonly color: ChipProps['color'];
};

const statusAppearances: Readonly<Record<OrderStatus, StatusAppearance>> = {
  en_attente_paiement: { icon: <HourglassTopRounded />, color: 'secondary' },
};

type OrderStatusChipProps = {
  readonly status: OrderStatus;
  readonly label: string;
};

export function OrderStatusChip({ status, label }: OrderStatusChipProps) {
  const { icon, color } = statusAppearances[status];
  return <Chip icon={icon} label={label} color={color} variant="outlined" />;
}
