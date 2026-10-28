import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { ReactNode } from 'react';

type FactItemProps = {
  readonly label: string;
  readonly icon?: ReactNode;
  readonly children: ReactNode;
};

export function FactItem({ label, icon, children }: FactItemProps) {
  return (
    <Stack spacing={0.5}>
      <Stack component="dt" direction="row" spacing={1} alignItems="center">
        {icon !== undefined && (
          <Stack component="span" aria-hidden="true" color="primary.main">
            {icon}
          </Stack>
        )}
        <Typography component="span" variant="body2" color="text.secondary">
          {label}
        </Typography>
      </Stack>
      <Typography component="dd" variant="body1" margin={0} fontWeight={500}>
        {children}
      </Typography>
    </Stack>
  );
}
