import Stack from '@mui/material/Stack';
import type { ReactNode } from 'react';

type FactListProps = {
  readonly children: ReactNode;
};

export function FactList({ children }: FactListProps) {
  return (
    <Stack component="dl" spacing={2} margin={0}>
      {children}
    </Stack>
  );
}
