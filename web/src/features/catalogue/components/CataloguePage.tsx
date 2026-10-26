import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { usePageIdentity } from '../../../navigation/usePageIdentity';
import { LiveStatus } from '../../../ui/components/LiveStatus';
import { PageHeading } from '../../../ui/components/PageHeading';
import { useCataloguePage } from '../hooks/useCataloguePage';
import { CatalogueContent } from './CatalogueContent';

export function CataloguePage() {
  const { t } = useTranslation();
  const headingRef = usePageIdentity(t('catalogue.documentTitle'));
  const { announcement, ...content } = useCataloguePage();
  return (
    <Stack spacing={{ xs: 3, md: 5 }}>
      <PageHeading
        headingRef={headingRef}
        title={t('catalogue.title')}
        subtitle={t('catalogue.subtitle')}
      />
      <LiveStatus message={announcement} />
      <CatalogueContent {...content} />
    </Stack>
  );
}
