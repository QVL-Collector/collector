import type { PaginationProps } from '@mui/material/Pagination';
import { useCallback } from 'react';
import { useTranslation } from 'react-i18next';

export type PaginationItemLabel = NonNullable<PaginationProps['getItemAriaLabel']>;

export function usePaginationItemLabel(): PaginationItemLabel {
  const { t } = useTranslation();
  return useCallback<PaginationItemLabel>(
    (type, page, selected) => {
      if (type === 'first') {
        return t('common.pagination.first');
      }
      if (type === 'last') {
        return t('common.pagination.last');
      }
      if (type === 'previous') {
        return t('common.pagination.previous');
      }
      if (type === 'next') {
        return t('common.pagination.next');
      }
      return selected
        ? t('common.pagination.currentPage', { page })
        : t('common.pagination.page', { page });
    },
    [t],
  );
}
