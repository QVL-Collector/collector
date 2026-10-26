import CircularProgress from '@mui/material/CircularProgress';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { revealSx } from '../animations/revealSx';

type PendingStateProps = {
  readonly message: string;
};

export function PendingState({ message }: PendingStateProps) {
  return (
    <Stack alignItems="center" spacing={2} paddingY={{ xs: 6, md: 10 }} sx={revealSx(0)}>
      <CircularProgress
        aria-hidden="true"
        size="2.5rem"
        sx={{ '@media (prefers-reduced-motion: reduce)': { animationDuration: '4s' } }}
      />
      <Typography role="status" aria-live="polite" color="text.secondary" align="center">
        {message}
      </Typography>
    </Stack>
  );
}
