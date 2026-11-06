import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { usePaginationItemLabel } from '../../../i18n/usePaginationItemLabel';
import { revealSx } from '../../../ui/animations/revealSx';
import { PageNavigation } from '../../../ui/components/PageNavigation';
import type { ReviewArticle } from '../domain/reviewArticle';
import type { DecisionListener } from '../hooks/useReviewDecision';
import { ReviewArticleCard } from './ReviewArticleCard';

type ReviewQueueResultsProps = {
  readonly articles: readonly ReviewArticle[];
  readonly page: number;
  readonly pageCount: number;
  readonly isRefreshing: boolean;
  readonly onPageChange: (page: number) => void;
  readonly onDecided: DecisionListener;
};

export function ReviewQueueResults({
  articles,
  page,
  pageCount,
  isRefreshing,
  onPageChange,
  onDecided,
}: ReviewQueueResultsProps) {
  const { t } = useTranslation();
  const getItemAriaLabel = usePaginationItemLabel();
  return (
    <Stack spacing={{ xs: 4, md: 5 }}>
      <Stack
        component="ul"
        aria-label={t('admin.review.listLabel')}
        aria-busy={isRefreshing}
        spacing={{ xs: 2, md: 3 }}
        margin={0}
        padding={0}
        sx={{
          listStyle: 'none',
          opacity: isRefreshing ? 0.55 : 1,
          transition: (theme) => theme.transitions.create('opacity'),
        }}
      >
        {articles.map((article, index) => (
          <Stack key={article.id} component="li" sx={revealSx(index)}>
            <ReviewArticleCard article={article} onDecided={onDecided} />
          </Stack>
        ))}
      </Stack>
      {pageCount > 1 && (
        <PageNavigation
          label={t('admin.review.paginationLabel')}
          page={page}
          pageCount={pageCount}
          onPageChange={onPageChange}
          getItemAriaLabel={getItemAriaLabel}
        />
      )}
    </Stack>
  );
}
