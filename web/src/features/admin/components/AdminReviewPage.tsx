import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import { LiveStatus } from '../../../ui/components/LiveStatus';
import { PageHeading } from '../../../ui/components/PageHeading';
import { useAdminReviewPage } from '../hooks/useAdminReviewPage';
import { ReviewQueueContent } from './ReviewQueueContent';

export function AdminReviewPage() {
  const { t } = useTranslation();
  const listHeadingId = useId();
  const { headingRef, listHeadingRef, listAnnouncement, decisionAnnouncement, ...content } =
    useAdminReviewPage();
  return (
    <Stack spacing={{ xs: 3, md: 5 }}>
      <PageHeading
        headingRef={headingRef}
        title={t('admin.review.title')}
        subtitle={t('admin.review.subtitle')}
      />
      <LiveStatus message={listAnnouncement} />
      <LiveStatus message={decisionAnnouncement} />
      <Stack component="section" spacing={3} aria-labelledby={listHeadingId}>
        <Typography
          ref={listHeadingRef}
          id={listHeadingId}
          component="h2"
          variant="h2"
          tabIndex={-1}
        >
          {t('admin.review.listTitle')}
        </Typography>
        <ReviewQueueContent {...content} />
      </Stack>
    </Stack>
  );
}
