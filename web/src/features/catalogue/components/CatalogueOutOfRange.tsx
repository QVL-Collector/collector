import SearchOffOutlined from '@mui/icons-material/SearchOffOutlined';
import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { EmptyState } from '../../../ui/components/EmptyState';

type CatalogueOutOfRangeProps = {
  readonly onBackToFirstPage: () => void;
};

export function CatalogueOutOfRange({ onBackToFirstPage }: CatalogueOutOfRangeProps) {
  const { t } = useTranslation();
  return (
    <EmptyState
      icon={<SearchOffOutlined fontSize="inherit" />}
      title={t('catalogue.outOfRange.title')}
      description={t('catalogue.outOfRange.description')}
      action={
        <Button variant="contained" onClick={onBackToFirstPage}>
          {t('catalogue.outOfRange.action')}
        </Button>
      }
    />
  );
}
