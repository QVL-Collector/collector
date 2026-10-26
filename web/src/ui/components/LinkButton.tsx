import Button from '@mui/material/Button';
import type { ReactNode } from 'react';
import { Link as RouterLink } from 'react-router';

type LinkButtonProps = {
  readonly to: string;
  readonly label: string;
  readonly icon?: ReactNode;
  readonly variant: 'contained' | 'outlined' | 'text';
};

export function LinkButton({ to, label, icon, variant }: LinkButtonProps) {
  return (
    <Button
      component={RouterLink}
      to={to}
      variant={variant}
      startIcon={icon}
      sx={{ minHeight: '2.75rem' }}
    >
      {label}
    </Button>
  );
}
