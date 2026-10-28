import CategoryOutlined from '@mui/icons-material/CategoryOutlined';
import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { EmptyState } from '../../../ui/components/EmptyState';

type CategoryEmptyStateProps = {
  readonly onShowAllCategories: () => void;
};

export function CategoryEmptyState({ onShowAllCategories }: CategoryEmptyStateProps) {
  const { t } = useTranslation();
  return (
    <EmptyState
      icon={<CategoryOutlined fontSize="inherit" />}
      title={t('catalogue.emptyCategory.title')}
      description={t('catalogue.emptyCategory.description')}
      action={
        <Button variant="contained" onClick={onShowAllCategories}>
          {t('catalogue.showAllCategories')}
        </Button>
      }
    />
  );
}
