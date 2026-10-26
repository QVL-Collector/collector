import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { usePageIdentity } from '../../navigation/usePageIdentity';
import { PageHeading } from '../../ui/components/PageHeading';
import { BackToCatalogueButton } from './BackToCatalogueButton';

export function AccessDeniedPage() {
  const { t } = useTranslation();
  const headingRef = usePageIdentity(t('accessDenied.documentTitle'));
  return (
    <Stack spacing={4} alignItems="flex-start">
      <PageHeading
        headingRef={headingRef}
        title={t('accessDenied.title')}
        subtitle={t('accessDenied.description')}
      />
      <BackToCatalogueButton />
    </Stack>
  );
}
