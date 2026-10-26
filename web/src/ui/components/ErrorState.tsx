import Alert from '@mui/material/Alert';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { ReactNode } from 'react';
import { revealSx } from '../animations/revealSx';

type ErrorStateProps = {
  readonly title: string;
  readonly description: string;
  readonly action?: ReactNode;
};

export function ErrorState({ title, description, action }: ErrorStateProps) {
  return (
    <Stack alignItems="center" paddingY={{ xs: 4, md: 8 }} sx={revealSx(0)}>
      <Alert severity="error" variant="outlined" sx={{ width: '100%', maxWidth: '40rem' }}>
        <Stack spacing={2} alignItems="flex-start">
          <Stack spacing={0.5}>
            <Typography component="h2" variant="subtitle1">
              {title}
            </Typography>
            <Typography variant="body2">{description}</Typography>
          </Stack>
          {action}
        </Stack>
      </Alert>
    </Stack>
  );
}
