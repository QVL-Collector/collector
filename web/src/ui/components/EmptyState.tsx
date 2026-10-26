import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { ReactNode } from 'react';
import { revealSx } from '../animations/revealSx';

type EmptyStateProps = {
  readonly title: string;
  readonly description: string;
  readonly icon?: ReactNode;
  readonly action?: ReactNode;
};

export function EmptyState({ title, description, icon, action }: EmptyStateProps) {
  return (
    <Stack
      alignItems="center"
      spacing={2}
      paddingY={{ xs: 6, md: 10 }}
      paddingX={2}
      sx={revealSx(0)}
    >
      {icon !== undefined && (
        <Stack
          alignItems="center"
          justifyContent="center"
          aria-hidden="true"
          color="primary.main"
          sx={{ fontSize: '3.5rem' }}
        >
          {icon}
        </Stack>
      )}
      <Typography component="h2" variant="h2" align="center">
        {title}
      </Typography>
      <Typography variant="body1" color="text.secondary" align="center" maxWidth="32rem">
        {description}
      </Typography>
      {action}
    </Stack>
  );
}
