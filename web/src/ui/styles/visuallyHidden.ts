import type { SxProps, Theme } from '@mui/material/styles';

export const visuallyHidden: SxProps<Theme> = {
  position: 'absolute',
  width: '0.0625rem',
  height: '0.0625rem',
  margin: '-0.0625rem',
  padding: 0,
  border: 0,
  overflow: 'hidden',
  clip: 'rect(0 0 0 0)',
  whiteSpace: 'nowrap',
};
