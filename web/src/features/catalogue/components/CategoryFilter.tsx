import TextField from '@mui/material/TextField';
import { useTranslation } from 'react-i18next';
import { allCategoriesValue, type CategoryFilterModel } from '../hooks/useCategoryFilter';

type CategoryFilterProps = {
  readonly filter: CategoryFilterModel;
};

const categoryFilterId = 'filtre-categorie';

export function CategoryFilter({ filter }: CategoryFilterProps) {
  const { t } = useTranslation();
  return (
    <TextField
      select
      id={categoryFilterId}
      label={t('catalogue.filter.label')}
      value={filter.value}
      onChange={(event) => {
        filter.onChange(event.target.value);
      }}
      helperText={filter.unavailable ? t('catalogue.filter.unavailable') : undefined}
      fullWidth
      slotProps={{ select: { native: true }, inputLabel: { shrink: true } }}
    >
      <option value={allCategoriesValue}>{t('catalogue.filter.all')}</option>
      {filter.options.map((category) => (
        <option key={category.id} value={category.id}>
          {category.label}
        </option>
      ))}
    </TextField>
  );
}
