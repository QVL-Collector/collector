import Container from '@mui/material/Container';
import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { Outlet } from 'react-router';
import { SkipLink } from '../../ui/components/SkipLink';
import { AppHeader } from './AppHeader';

const mainContentId = 'contenu-principal';

export function AppLayout() {
  const { t } = useTranslation();
  return (
    <Stack minHeight="100dvh">
      <SkipLink targetId={mainContentId} label={t('app.skipToContent')} />
      <AppHeader />
      <Container component="main" id={mainContentId} tabIndex={-1} maxWidth="lg">
        <Stack paddingY={{ xs: 3, md: 6 }}>
          <Outlet />
        </Stack>
      </Container>
    </Stack>
  );
}
