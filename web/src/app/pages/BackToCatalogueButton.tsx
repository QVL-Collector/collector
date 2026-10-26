import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { Link as RouterLink } from 'react-router';

export function BackToCatalogueButton() {
  const { t } = useTranslation();
  return (
    <Button component={RouterLink} to="/" variant="contained">
      {t('common.backToCatalogue')}
    </Button>
  );
}
