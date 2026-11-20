import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import { LiveStatus } from '../../../ui/components/LiveStatus';
import { PageHeading } from '../../../ui/components/PageHeading';
import { useMyOrdersPage } from '../hooks/useMyOrdersPage';
import { MyOrdersContent } from './MyOrdersContent';

export function MyOrdersPage() {
  const { t } = useTranslation();
  const listHeadingId = useId();
  const { headingRef, listAnnouncement, expiryAnnouncement, ...content } = useMyOrdersPage();
  return (
    <Stack spacing={{ xs: 3, md: 5 }}>
      <PageHeading
        headingRef={headingRef}
        title={t('orders.list.title')}
        subtitle={t('orders.list.subtitle')}
      />
      <LiveStatus message={listAnnouncement} />
      <LiveStatus message={expiryAnnouncement} />
      <Stack component="section" spacing={3} aria-labelledby={listHeadingId}>
        <Typography id={listHeadingId} component="h2" variant="h2">
          {t('orders.list.listTitle')}
        </Typography>
        <MyOrdersContent {...content} />
      </Stack>
    </Stack>
  );
}
