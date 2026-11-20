import type { SxProps, Theme } from '@mui/material/styles';

export const liftOnHoverSx = ((theme: Theme) => ({
  transition: theme.transitions.create(['transform', 'box-shadow']),
  '&:hover, &:focus-within': { transform: 'translateY(-0.125rem)', boxShadow: 3 },
  '@media (prefers-reduced-motion: reduce)': {
    transition: 'none',
    '&:hover, &:focus-within': { transform: 'none' },
  },
})) satisfies SxProps<Theme>;
