import TextField from '@mui/material/TextField';
import { useTranslation } from 'react-i18next';
import type { CategoryOptions } from '../hooks/useCategoryOptions';
import type { ArticleFieldModel } from '../hooks/useArticleFieldModels';

type ArticleCategoryFieldProps = {
  readonly field: ArticleFieldModel;
  readonly categories: CategoryOptions;
};

export function ArticleCategoryField({ field, categories }: ArticleCategoryFieldProps) {
  const { t } = useTranslation();
  const hint = categories.unavailable ? t('seller.form.categoriesUnavailable') : undefined;
  return (
    <TextField
      select
      id={field.id}
      label={t('seller.form.category')}
      value={field.value}
      onChange={(event) => {
        field.onChange(event.target.value);
      }}
      inputRef={field.inputRef}
      error={field.error !== null}
      helperText={field.error ?? hint}
      fullWidth
      slotProps={{ select: { native: true }, inputLabel: { shrink: true } }}
    >
      <option value="">{t('seller.form.noCategory')}</option>
      {categories.options.map((category) => (
        <option key={category.id} value={category.id}>
          {category.label}
        </option>
      ))}
    </TextField>
  );
}
