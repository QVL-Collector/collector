import AppBar from '@mui/material/AppBar';
import Container from '@mui/material/Container';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import { alpha } from '@mui/material/styles';
import { useTranslation } from 'react-i18next';
import { Link as RouterLink } from 'react-router';
import { LanguageSwitcher } from '../../ui/components/LanguageSwitcher';
import { AccountControl } from './AccountControl';
import { useLanguageSwitcher } from './useLanguageSwitcher';

export function AppHeader() {
  const { t } = useTranslation();
  const { label, language, options, changeLanguage } = useLanguageSwitcher();
  return (
    <AppBar
      position="sticky"
      color="inherit"
      elevation={0}
      sx={(theme) => ({
        backgroundColor: alpha(theme.palette.background.default, 0.88),
        backdropFilter: 'blur(0.75rem)',
        borderBottom: `0.0625rem solid ${theme.palette.divider}`,
      })}
    >
      <Container maxWidth="lg">
        <Stack
          direction="row"
          alignItems="center"
          justifyContent="space-between"
          spacing={2}
          minHeight={{ xs: '3.5rem', md: '4.5rem' }}
        >
          <Link
            component={RouterLink}
            to="/"
            variant="h6"
            color="primary"
            underline="none"
            aria-label={t('app.homeLink')}
            sx={{ fontWeight: 700, letterSpacing: '-0.01em' }}
          >
            {t('app.name')}
          </Link>
          <Stack direction="row" alignItems="center" spacing={{ xs: 1, sm: 2 }} minWidth={0}>
            <LanguageSwitcher
              label={label}
              value={language}
              options={options}
              onChange={changeLanguage}
            />
            <AccountControl />
          </Stack>
        </Stack>
      </Container>
    </AppBar>
  );
}
