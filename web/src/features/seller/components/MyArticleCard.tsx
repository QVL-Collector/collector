import Card from '@mui/material/Card';
import CardActionArea from '@mui/material/CardActionArea';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { Link as RouterLink } from 'react-router';
import type { SellerArticle } from '../domain/sellerArticle';
import { useMyArticleCard } from '../hooks/useMyArticleCard';
import { ArticleStateChip } from './ArticleStateChip';

type MyArticleCardProps = {
  readonly article: SellerArticle;
};

export function MyArticleCard({ article }: MyArticleCardProps) {
  const { path, title, priceLabel, updatedLabel, status } = useMyArticleCard(article);
  return (
    <Card
      component="article"
      variant="outlined"
      sx={{
        height: '100%',
        transition: (theme) => theme.transitions.create(['transform', 'box-shadow']),
        '&:hover, &:focus-within': { transform: 'translateY(-0.125rem)', boxShadow: 3 },
      }}
    >
      <CardActionArea component={RouterLink} to={path} sx={{ height: '100%' }}>
        <Stack spacing={1.5} padding={2.5} height="100%" justifyContent="space-between">
          <Stack spacing={1.5} alignItems="flex-start">
            <ArticleStateChip state={status.state} label={status.label} />
            <Typography
              component="h3"
              variant="h6"
              sx={{
                display: '-webkit-box',
                WebkitLineClamp: 3,
                WebkitBoxOrient: 'vertical',
                overflow: 'hidden',
                overflowWrap: 'anywhere',
              }}
            >
              {title}
            </Typography>
            {status.reason !== null && (
              <Typography variant="body2" color="text.secondary">
                {status.reason}
              </Typography>
            )}
          </Stack>
          <Stack spacing={0.5}>
            <Typography component="p" variant="subtitle1" color="primary">
              {priceLabel}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              {updatedLabel}
            </Typography>
          </Stack>
        </Stack>
      </CardActionArea>
    </Card>
  );
}
