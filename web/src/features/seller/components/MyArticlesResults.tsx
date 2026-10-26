import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { usePaginationItemLabel } from '../../../i18n/usePaginationItemLabel';
import { CardGrid } from '../../../ui/components/CardGrid';
import { CardGridItem } from '../../../ui/components/CardGridItem';
import { PageNavigation } from '../../../ui/components/PageNavigation';
import type { SellerArticle } from '../domain/sellerArticle';
import { MyArticleCard } from './MyArticleCard';

type MyArticlesResultsProps = {
  readonly articles: readonly SellerArticle[];
  readonly page: number;
  readonly pageCount: number;
  readonly isRefreshing: boolean;
  readonly onPageChange: (page: number) => void;
};

export function MyArticlesResults({
  articles,
  page,
  pageCount,
  isRefreshing,
  onPageChange,
}: MyArticlesResultsProps) {
  const { t } = useTranslation();
  const getItemAriaLabel = usePaginationItemLabel();
  return (
    <Stack spacing={{ xs: 4, md: 5 }}>
      <Stack
        aria-busy={isRefreshing}
        sx={{
          opacity: isRefreshing ? 0.55 : 1,
          transition: (theme) => theme.transitions.create('opacity'),
        }}
      >
        <CardGrid label={t('seller.list.title')}>
          {articles.map((article, index) => (
            <CardGridItem key={article.id} index={index}>
              <MyArticleCard article={article} />
            </CardGridItem>
          ))}
        </CardGrid>
      </Stack>
      {pageCount > 1 && (
        <PageNavigation
          label={t('seller.list.paginationLabel')}
          page={page}
          pageCount={pageCount}
          onPageChange={onPageChange}
          getItemAriaLabel={getItemAriaLabel}
        />
      )}
    </Stack>
  );
}
