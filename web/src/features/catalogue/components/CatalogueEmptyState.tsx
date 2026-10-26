import Inventory2Outlined from '@mui/icons-material/Inventory2Outlined';
import { useTranslation } from 'react-i18next';
import { EmptyState } from '../../../ui/components/EmptyState';

export function CatalogueEmptyState() {
  const { t } = useTranslation();
  return (
    <EmptyState
      icon={<Inventory2Outlined fontSize="inherit" />}
      title={t('catalogue.empty.title')}
      description={t('catalogue.empty.description')}
    />
  );
}
