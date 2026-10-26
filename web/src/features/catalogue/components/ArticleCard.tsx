import Card from '@mui/material/Card';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import type { CatalogueArticle } from '../domain/catalogue';

type ArticleCardProps = {
  readonly article: CatalogueArticle;
  readonly priceLabel: string;
};

export function ArticleCard({ article, priceLabel }: ArticleCardProps) {
  return (
    <Card component="article" variant="outlined" sx={{ height: '100%' }}>
      <Stack spacing={2} padding={2.5} height="100%" justifyContent="space-between">
        <Stack spacing={1.5} alignItems="flex-start">
          <Chip label={article.category.label} size="small" variant="outlined" color="primary" />
          <Typography
            component="h2"
            variant="h6"
            sx={{
              display: '-webkit-box',
              WebkitLineClamp: 3,
              WebkitBoxOrient: 'vertical',
              overflow: 'hidden',
              overflowWrap: 'anywhere',
            }}
          >
            {article.title}
          </Typography>
        </Stack>
        <Typography component="p" variant="subtitle1" color="primary">
          {priceLabel}
        </Typography>
      </Stack>
    </Card>
  );
}
