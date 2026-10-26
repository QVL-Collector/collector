import Pagination, { type PaginationProps } from '@mui/material/Pagination';
import Stack from '@mui/material/Stack';
import type { ChangeEvent } from 'react';
import { useIsCompactScreen } from '../hooks/useIsCompactScreen';

type PageNavigationProps = {
  readonly label: string;
  readonly page: number;
  readonly pageCount: number;
  readonly onPageChange: (page: number) => void;
  readonly getItemAriaLabel: NonNullable<PaginationProps['getItemAriaLabel']>;
};

export function PageNavigation({
  label,
  page,
  pageCount,
  onPageChange,
  getItemAriaLabel,
}: PageNavigationProps) {
  const isCompact = useIsCompactScreen();
  const handleChange = (_event: ChangeEvent<unknown>, next: number) => {
    onPageChange(next);
  };
  return (
    <Stack alignItems="center">
      <Pagination
        aria-label={label}
        page={page}
        count={pageCount}
        onChange={handleChange}
        getItemAriaLabel={getItemAriaLabel}
        color="primary"
        shape="rounded"
        size={isCompact ? 'medium' : 'large'}
        siblingCount={isCompact ? 0 : 1}
        boundaryCount={1}
      />
    </Stack>
  );
}
