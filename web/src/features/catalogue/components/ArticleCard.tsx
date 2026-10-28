import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { CardTitle } from '../../../ui/components/CardTitle';
import { LinkCard } from '../../../ui/components/LinkCard';
import type { CatalogueArticle } from '../domain/catalogue';
import type { ArticleCardLink } from '../hooks/useArticleCardLink';

type ArticleCardProps = {
  readonly article: CatalogueArticle;
  readonly priceLabel: string;
  readonly link: ArticleCardLink;
};

export function ArticleCard({ article, priceLabel, link }: ArticleCardProps) {
  return (
    <LinkCard to={link.to} state={link.state}>
      <Stack spacing={2} padding={2.5} height="100%" justifyContent="space-between">
        <Stack spacing={1.5} alignItems="flex-start">
          <Chip label={article.category.label} size="small" variant="outlined" color="primary" />
          <CardTitle level="h2">{article.title}</CardTitle>
        </Stack>
        <Typography component="p" variant="subtitle1" color="primary">
          {priceLabel}
        </Typography>
      </Stack>
    </LinkCard>
  );
}
