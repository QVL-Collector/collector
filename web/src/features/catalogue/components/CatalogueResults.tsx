import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { usePaginationItemLabel } from '../../../i18n/usePaginationItemLabel';
import { PageNavigation } from '../../../ui/components/PageNavigation';
import type { CatalogueArticle } from '../domain/catalogue';
import { CatalogueGrid } from './CatalogueGrid';

type CatalogueResultsProps = {
  readonly articles: readonly CatalogueArticle[];
  readonly page: number;
  readonly pageCount: number;
  readonly isRefreshing: boolean;
  readonly onPageChange: (page: number) => void;
};

export function CatalogueResults({
  articles,
  page,
  pageCount,
  isRefreshing,
  onPageChange,
}: CatalogueResultsProps) {
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
        <CatalogueGrid articles={articles} />
      </Stack>
      {pageCount > 1 && (
        <PageNavigation
          label={t('catalogue.paginationLabel')}
          page={page}
          pageCount={pageCount}
          onPageChange={onPageChange}
          getItemAriaLabel={getItemAriaLabel}
        />
      )}
    </Stack>
  );
}
