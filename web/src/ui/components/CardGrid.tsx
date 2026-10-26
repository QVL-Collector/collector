import Grid from '@mui/material/Grid';
import type { ReactNode } from 'react';

type CardGridProps = {
  readonly label?: string;
  readonly hiddenFromAssistiveTechnologies?: boolean;
  readonly children: ReactNode;
};

export function CardGrid({
  label,
  hiddenFromAssistiveTechnologies = false,
  children,
}: CardGridProps) {
  return (
    <Grid
      container
      component="ul"
      spacing={{ xs: 2, md: 3 }}
      aria-label={label}
      aria-hidden={hiddenFromAssistiveTechnologies || undefined}
      sx={{ listStyle: 'none', margin: 0, padding: 0 }}
    >
      {children}
    </Grid>
  );
}
