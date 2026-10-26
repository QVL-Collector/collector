import Grid from '@mui/material/Grid';
import type { ReactNode } from 'react';

type CardGridProps = {
  readonly label?: string;
  readonly hidden?: boolean;
  readonly children: ReactNode;
};

export function CardGrid({ label, hidden = false, children }: CardGridProps) {
  return (
    <Grid
      container
      component="ul"
      spacing={{ xs: 2, md: 3 }}
      aria-label={label}
      aria-hidden={hidden || undefined}
      sx={{ listStyle: 'none', margin: 0, padding: 0 }}
    >
      {children}
    </Grid>
  );
}
