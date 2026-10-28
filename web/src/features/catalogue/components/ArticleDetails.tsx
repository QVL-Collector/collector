import Stack from '@mui/material/Stack';
import type { PublishedArticle } from '../domain/catalogue';
import { usePublishedArticleSummary } from '../hooks/usePublishedArticleSummary';
import { ArticleDescription } from './ArticleDescription';
import { ArticleSummaryCard } from './ArticleSummaryCard';

type ArticleDetailsProps = {
  readonly article: PublishedArticle;
};

export function ArticleDetails({ article }: ArticleDetailsProps) {
  const summary = usePublishedArticleSummary(article);
  return (
    <Stack
      direction={{ xs: 'column', md: 'row-reverse' }}
      spacing={{ xs: 4, md: 6 }}
      alignItems={{ xs: 'stretch', md: 'flex-start' }}
    >
      <Stack
        width={{ xs: '100%', md: '22rem' }}
        flexShrink={0}
        position={{ md: 'sticky' }}
        top={{ md: '6rem' }}
      >
        <ArticleSummaryCard summary={summary} />
      </Stack>
      <ArticleDescription description={summary.description} />
    </Stack>
  );
}
