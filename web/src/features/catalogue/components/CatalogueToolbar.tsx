import Stack from '@mui/material/Stack';
import type { CategoryFilterModel } from '../hooks/useCategoryFilter';
import { CategoryFilter } from './CategoryFilter';

type CatalogueToolbarProps = {
  readonly filter: CategoryFilterModel;
};

export function CatalogueToolbar({ filter }: CatalogueToolbarProps) {
  return (
    <Stack direction="row" width="100%" maxWidth={{ xs: '100%', sm: '20rem' }}>
      <CategoryFilter filter={filter} />
    </Stack>
  );
}
