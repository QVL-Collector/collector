import RefreshRounded from '@mui/icons-material/RefreshRounded';
import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { ErrorState } from '../../../ui/components/ErrorState';

type CatalogueErrorStateProps = {
  readonly onRetry: () => void;
};

export function CatalogueErrorState({ onRetry }: CatalogueErrorStateProps) {
  const { t } = useTranslation();
  return (
    <ErrorState
      title={t('catalogue.error.title')}
      description={t('catalogue.error.description')}
      action={
        <Button variant="outlined" color="error" startIcon={<RefreshRounded />} onClick={onRetry}>
          {t('catalogue.error.retry')}
        </Button>
      }
    />
  );
}
