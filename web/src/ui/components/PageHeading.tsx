import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { Ref } from 'react';

type PageHeadingProps = {
  readonly title: string;
  readonly subtitle?: string;
  readonly headingRef?: Ref<HTMLHeadingElement>;
};

export function PageHeading({ title, subtitle, headingRef }: PageHeadingProps) {
  return (
    <Stack spacing={1} maxWidth="48rem">
      <Typography
        ref={headingRef}
        component="h1"
        variant="h1"
        tabIndex={-1}
        sx={{ overflowWrap: 'anywhere' }}
      >
        {title}
      </Typography>
      {subtitle !== undefined && (
        <Typography variant="body1" color="text.secondary">
          {subtitle}
        </Typography>
      )}
    </Stack>
  );
}
