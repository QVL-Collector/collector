import Card from '@mui/material/Card';
import CardActionArea from '@mui/material/CardActionArea';
import type { ReactNode } from 'react';
import { Link as RouterLink } from 'react-router';

type LinkCardProps = {
  readonly to: string;
  readonly state?: unknown;
  readonly children: ReactNode;
};

export function LinkCard({ to, state, children }: LinkCardProps) {
  return (
    <Card
      component="article"
      variant="outlined"
      sx={{
        height: '100%',
        transition: (theme) => theme.transitions.create(['transform', 'box-shadow']),
        '&:hover, &:focus-within': { transform: 'translateY(-0.125rem)', boxShadow: 3 },
        '@media (prefers-reduced-motion: reduce)': {
          transition: 'none',
          '&:hover, &:focus-within': { transform: 'none' },
        },
      }}
    >
      <CardActionArea component={RouterLink} to={to} state={state} sx={{ height: '100%' }}>
        {children}
      </CardActionArea>
    </Card>
  );
}
