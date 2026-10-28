import SaveRounded from '@mui/icons-material/SaveRounded';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';
import { LiveStatus } from '../../../ui/components/LiveStatus';
import { articleLimits } from '../domain/sellerArticle';
import type { ArticleEditorModel } from '../hooks/useArticleEditor';
import type { CategoryOptions } from '../hooks/useCategoryOptions';
import { ArticleCategoryField } from './ArticleCategoryField';
import { ArticleFormFeedback } from './ArticleFormFeedback';
import { ArticleTextField } from './ArticleTextField';

type ArticleFormProps = {
  readonly editor: ArticleEditorModel;
  readonly categories: CategoryOptions;
};

export function ArticleForm({ editor, categories }: ArticleFormProps) {
  const { t } = useTranslation();
  const { fields, feedback, savedAnnouncement, isSaving, mode, submit, reloadLatest } = editor;
  const { titleMaxLength, descriptionMaxLength } = articleLimits;
  return (
    <Stack
      component="form"
      noValidate
      onSubmit={submit}
      aria-label={t('seller.form.label')}
      spacing={3}
    >
      <Typography variant="body2" color="text.secondary">
        {t('seller.form.hint')}
      </Typography>
      <ArticleTextField
        field={fields.title}
        label={t('seller.form.title')}
        hint={t('seller.form.characterCount', {
          length: fields.title.value.length,
          max: titleMaxLength,
        })}
        maxLength={titleMaxLength}
      />
      <ArticleTextField
        field={fields.description}
        label={t('seller.form.description')}
        hint={t('seller.form.characterCount', {
          length: fields.description.value.length,
          max: descriptionMaxLength,
        })}
        maxLength={descriptionMaxLength}
        multiline
      />
      <ArticleCategoryField field={fields.categoryId} categories={categories} />
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={3}>
        <ArticleTextField
          field={fields.price}
          label={t('seller.form.price')}
          hint={t('seller.form.amountHint')}
          inputMode="decimal"
        />
        <ArticleTextField
          field={fields.shippingCost}
          label={t('seller.form.shippingCost')}
          hint={t('seller.form.amountHint')}
          inputMode="decimal"
        />
      </Stack>
      <ArticleFormFeedback feedback={feedback} onReload={reloadLatest} />
      <LiveStatus message={savedAnnouncement} />
      <Stack alignItems={{ xs: 'stretch', sm: 'flex-start' }}>
        <Button
          type="submit"
          variant="contained"
          startIcon={<SaveRounded />}
          loading={isSaving}
          loadingPosition="start"
          sx={{ minHeight: '2.75rem' }}
        >
          {mode === 'create' ? t('seller.form.create') : t('seller.form.save')}
        </Button>
      </Stack>
    </Stack>
  );
}
