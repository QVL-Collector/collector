import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import type { ReactNode } from 'react';

type ResponsiveActionButtonProps = {
  readonly label: string;
  readonly icon: ReactNode;
  readonly compact: boolean;
  readonly loading: boolean;
  readonly variant: 'contained' | 'outlined' | 'text';
  readonly onClick: () => void;
};

export function ResponsiveActionButton({
  label,
  icon,
  compact,
  loading,
  variant,
  onClick,
}: ResponsiveActionButtonProps) {
  if (compact) {
    return (
      <IconButton
        aria-label={label}
        color="primary"
        size="large"
        loading={loading}
        onClick={onClick}
      >
        {icon}
      </IconButton>
    );
  }
  return (
    <Button
      variant={variant}
      startIcon={icon}
      loading={loading}
      loadingPosition="start"
      onClick={onClick}
      sx={{ minHeight: '2.75rem' }}
    >
      {label}
    </Button>
  );
}
