import Grid from '@mui/material/Grid';
import type { ReactNode } from 'react';
import { revealSx } from '../animations/revealSx';

type CardGridItemProps = {
  readonly index: number;
  readonly children: ReactNode;
};

export function CardGridItem({ index, children }: CardGridItemProps) {
  return (
    <Grid component="li" size={{ xs: 12, sm: 6, md: 4, lg: 3 }} sx={revealSx(index)}>
      {children}
    </Grid>
  );
}
