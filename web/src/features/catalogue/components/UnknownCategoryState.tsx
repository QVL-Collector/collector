import SearchOffOutlined from '@mui/icons-material/SearchOffOutlined';
import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { EmptyState } from '../../../ui/components/EmptyState';

type UnknownCategoryStateProps = {
  readonly onShowAllCategories: () => void;
};

export function UnknownCategoryState({ onShowAllCategories }: UnknownCategoryStateProps) {
  const { t } = useTranslation();
  return (
    <EmptyState
      icon={<SearchOffOutlined fontSize="inherit" />}
      title={t('catalogue.unknownCategory.title')}
      description={t('catalogue.unknownCategory.description')}
      action={
        <Button variant="contained" onClick={onShowAllCategories}>
          {t('catalogue.showAllCategories')}
        </Button>
      }
    />
  );
}
