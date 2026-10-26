import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { usePageIdentity } from '../../navigation/usePageIdentity';
import { PageHeading } from '../../ui/components/PageHeading';
import { BackToCatalogueButton } from './BackToCatalogueButton';

export function NotFoundPage() {
  const { t } = useTranslation();
  const headingRef = usePageIdentity(t('notFound.documentTitle'));
  return (
    <Stack spacing={4} alignItems="flex-start">
      <PageHeading
        headingRef={headingRef}
        title={t('notFound.title')}
        subtitle={t('notFound.description')}
      />
      <BackToCatalogueButton />
    </Stack>
  );
}
