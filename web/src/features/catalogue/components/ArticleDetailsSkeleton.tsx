import Card from '@mui/material/Card';
import Skeleton from '@mui/material/Skeleton';
import Stack from '@mui/material/Stack';

const descriptionLineWidths = ['95%', '88%', '92%', '60%'] as const;

export function ArticleDetailsSkeleton() {
  return (
    <Stack
      aria-hidden="true"
      direction={{ xs: 'column', md: 'row-reverse' }}
      spacing={{ xs: 4, md: 6 }}
      alignItems={{ xs: 'stretch', md: 'flex-start' }}
    >
      <Card variant="outlined" sx={{ width: { xs: '100%', md: '22rem' }, flexShrink: 0 }}>
        <Stack spacing={2} padding={{ xs: 2.5, md: 3 }}>
          <Skeleton variant="text" width="5rem" />
          <Skeleton variant="rounded" width="7rem" height="2.75rem" />
          <Skeleton variant="text" width="6rem" height="2.5rem" />
          <Skeleton variant="rectangular" height="0.0625rem" />
          <Skeleton variant="text" width="70%" />
          <Skeleton variant="text" width="60%" />
        </Stack>
      </Card>
      <Stack spacing={1} flex={1} width="100%">
        <Skeleton variant="text" width="9rem" height="2.25rem" />
        {descriptionLineWidths.map((width) => (
          <Skeleton key={width} variant="text" width={width} />
        ))}
      </Stack>
    </Stack>
  );
}
