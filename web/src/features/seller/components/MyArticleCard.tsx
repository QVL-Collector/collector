import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { CardTitle } from '../../../ui/components/CardTitle';
import { LinkCard } from '../../../ui/components/LinkCard';
import type { SellerArticle } from '../domain/sellerArticle';
import { useMyArticleCard } from '../hooks/useMyArticleCard';
import { ArticleStateChip } from './ArticleStateChip';

type MyArticleCardProps = {
  readonly article: SellerArticle;
};

export function MyArticleCard({ article }: MyArticleCardProps) {
  const { path, title, priceLabel, updatedLabel, status } = useMyArticleCard(article);
  return (
    <LinkCard to={path}>
      <Stack spacing={1.5} padding={2.5} height="100%" justifyContent="space-between">
        <Stack spacing={1.5} alignItems="flex-start">
          <ArticleStateChip state={status.state} label={status.label} />
          <CardTitle level="h3">{title}</CardTitle>
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
    </LinkCard>
  );
}
