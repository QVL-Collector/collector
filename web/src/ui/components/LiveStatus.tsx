import Typography from '@mui/material/Typography';
import { visuallyHidden } from '../styles/visuallyHidden';

type LiveStatusProps = {
  readonly message: string;
};

export function LiveStatus({ message }: LiveStatusProps) {
  return (
    <Typography component="p" role="status" aria-live="polite" sx={visuallyHidden}>
      {message}
    </Typography>
  );
}
