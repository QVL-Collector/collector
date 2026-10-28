import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import type { ReactNode } from 'react';
import { Link as RouterLink } from 'react-router';

type SpaceLinkProps = {
  readonly to: string;
  readonly label: string;
  readonly icon: ReactNode;
  readonly compact: boolean;
};

export function SpaceLink({ to, label, icon, compact }: SpaceLinkProps) {
  if (compact) {
    return (
      <IconButton component={RouterLink} to={to} aria-label={label} color="primary" size="large">
        {icon}
      </IconButton>
    );
  }
  return (
    <Button
      component={RouterLink}
      to={to}
      variant="text"
      startIcon={icon}
      sx={{ minHeight: '2.75rem' }}
    >
      {label}
    </Button>
  );
}
