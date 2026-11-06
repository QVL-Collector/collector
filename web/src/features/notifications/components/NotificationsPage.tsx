import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import { LiveStatus } from '../../../ui/components/LiveStatus';
import { PageHeading } from '../../../ui/components/PageHeading';
import { useNotificationsPage } from '../hooks/useNotificationsPage';
import { NotificationsContent } from './NotificationsContent';

export function NotificationsPage() {
  const { t } = useTranslation();
  const listHeadingId = useId();
  const { headingRef, announcement, ...content } = useNotificationsPage();
  return (
    <Stack spacing={{ xs: 3, md: 5 }}>
      <PageHeading
        headingRef={headingRef}
        title={t('notifications.title')}
        subtitle={t('notifications.subtitle')}
      />
      <LiveStatus message={announcement} />
      <Stack component="section" spacing={3} aria-labelledby={listHeadingId}>
        <Typography id={listHeadingId} component="h2" variant="h2">
          {t('notifications.listTitle')}
        </Typography>
        <NotificationsContent {...content} />
      </Stack>
    </Stack>
  );
}
