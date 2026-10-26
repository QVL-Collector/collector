import Container from '@mui/material/Container';
import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { useDocumentTitle } from '../../ui/hooks/useDocumentTitle';
import { ErrorState } from '../../ui/components/ErrorState';
import { BackToCatalogueButton } from './BackToCatalogueButton';

export function RouteErrorPage() {
  const { t } = useTranslation();
  useDocumentTitle(t('routeError.documentTitle'));
  return (
    <Container component="main" maxWidth="lg">
      <Stack paddingY={{ xs: 3, md: 6 }}>
        <ErrorState
          title={t('routeError.title')}
          description={t('routeError.description')}
          action={<BackToCatalogueButton />}
        />
      </Stack>
    </Container>
  );
}
