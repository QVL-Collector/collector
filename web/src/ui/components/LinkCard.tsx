import Card from '@mui/material/Card';
import CardActionArea from '@mui/material/CardActionArea';
import type { ReactNode } from 'react';
import { Link as RouterLink } from 'react-router';
import { liftOnHoverSx } from '../animations/liftOnHoverSx';

type LinkCardProps = {
  readonly to: string;
  readonly state?: unknown;
  readonly children: ReactNode;
};

export function LinkCard({ to, state, children }: LinkCardProps) {
  return (
    <Card component="article" variant="outlined" sx={[liftOnHoverSx, { height: '100%' }]}>
      <CardActionArea component={RouterLink} to={to} state={state} sx={{ height: '100%' }}>
        {children}
      </CardActionArea>
    </Card>
  );
}
