import Avatar from '@mui/material/Avatar';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { revealSx } from '../animations/revealSx';
import { visuallyHidden } from '../styles/visuallyHidden';

type AccountBadgeProps = {
  readonly name: string;
  readonly detail: string;
  readonly initials: string;
  readonly compact: boolean;
};

export function AccountBadge({ name, detail, initials, compact }: AccountBadgeProps) {
  return (
    <Stack direction="row" alignItems="center" spacing={1.5} minWidth={0} sx={revealSx(0)}>
      <Avatar
        aria-hidden="true"
        sx={{
          width: '2.25rem',
          height: '2.25rem',
          fontSize: '0.875rem',
          fontWeight: 700,
          bgcolor: 'primary.main',
          color: 'primary.contrastText',
        }}
      >
        {initials}
      </Avatar>
      <Stack minWidth={0} sx={compact ? visuallyHidden : undefined}>
        <Typography variant="body2" fontWeight={500} noWrap>
          {name}
        </Typography>
        <Typography variant="caption" color="text.secondary" noWrap>
          {detail}
        </Typography>
      </Stack>
    </Stack>
  );
}
