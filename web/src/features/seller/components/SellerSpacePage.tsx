import AddRounded from '@mui/icons-material/AddRounded';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import { LinkButton } from '../../../ui/components/LinkButton';
import { LiveStatus } from '../../../ui/components/LiveStatus';
import { PageHeading } from '../../../ui/components/PageHeading';
import { useSellerSpacePage } from '../hooks/useSellerSpacePage';
import { newArticlePath } from '../sellerPaths';
import { MyArticlesContent } from './MyArticlesContent';

export function SellerSpacePage() {
  const { t } = useTranslation();
  const listHeadingId = useId();
  const { headingRef, title, welcome, listAnnouncement, stateAnnouncement, ...content } =
    useSellerSpacePage();
  return (
    <Stack spacing={{ xs: 3, md: 5 }}>
      <Stack
        direction={{ xs: 'column', sm: 'row' }}
        spacing={2}
        justifyContent="space-between"
        alignItems={{ xs: 'stretch', sm: 'flex-end' }}
      >
        <PageHeading headingRef={headingRef} title={title} subtitle={welcome} />
        <LinkButton
          to={newArticlePath}
          label={t('seller.newArticle')}
          icon={<AddRounded />}
          variant="contained"
        />
      </Stack>
      <LiveStatus message={listAnnouncement} />
      <LiveStatus message={stateAnnouncement} />
      <Stack component="section" spacing={3} aria-labelledby={listHeadingId}>
        <Typography id={listHeadingId} component="h2" variant="h2">
          {t('seller.list.title')}
        </Typography>
        <MyArticlesContent {...content} />
      </Stack>
    </Stack>
  );
}
