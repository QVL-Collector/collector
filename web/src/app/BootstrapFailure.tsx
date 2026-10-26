import Container from '@mui/material/Container';
import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { ErrorState } from '../ui/components/ErrorState';
import { useDocumentTitle } from '../ui/hooks/useDocumentTitle';

export function BootstrapFailure() {
  const { t } = useTranslation();
  useDocumentTitle(t('app.bootstrapFailure.title'));
  return (
    <Container component="main" maxWidth="lg">
      <Stack paddingY={{ xs: 3, md: 6 }}>
        <ErrorState
          title={t('app.bootstrapFailure.title')}
          description={t('app.bootstrapFailure.description')}
        />
      </Stack>
    </Container>
  );
}
