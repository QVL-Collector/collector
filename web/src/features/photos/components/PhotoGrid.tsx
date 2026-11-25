import Box from '@mui/material/Box';
import type { ReactNode } from 'react';

type PhotoGridProps = {
  readonly maxWidth?: string;
  readonly children: ReactNode;
};

export function PhotoGrid({ maxWidth = '40rem', children }: PhotoGridProps) {
  return (
    <Box
      display="grid"
      gridTemplateColumns="repeat(auto-fit, minmax(min(100%, 14rem), 1fr))"
      gap={{ xs: 2, md: 3 }}
      maxWidth={maxWidth}
    >
      {children}
    </Box>
  );
}
