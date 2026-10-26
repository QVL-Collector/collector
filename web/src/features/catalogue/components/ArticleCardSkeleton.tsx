import Card from '@mui/material/Card';
import Skeleton from '@mui/material/Skeleton';
import Stack from '@mui/material/Stack';

export function ArticleCardSkeleton() {
  return (
    <Card variant="outlined" sx={{ height: '100%' }}>
      <Stack spacing={2} padding={2.5}>
        <Skeleton variant="rounded" width="6rem" height="1.5rem" />
        <Stack spacing={0.5}>
          <Skeleton variant="text" width="90%" />
          <Skeleton variant="text" width="65%" />
        </Stack>
        <Skeleton variant="text" width="4.5rem" height="1.75rem" />
      </Stack>
    </Card>
  );
}
