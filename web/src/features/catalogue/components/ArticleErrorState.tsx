import RefreshRounded from '@mui/icons-material/RefreshRounded';
import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { ErrorState } from '../../../ui/components/ErrorState';

type ArticleErrorStateProps = {
  readonly onRetry: () => void;
};

export function ArticleErrorState({ onRetry }: ArticleErrorStateProps) {
  const { t } = useTranslation();
  return (
    <ErrorState
      title={t('article.error.title')}
      description={t('article.error.description')}
      action={
        <Button variant="outlined" color="error" startIcon={<RefreshRounded />} onClick={onRetry}>
          {t('article.error.retry')}
        </Button>
      }
    />
  );
}
