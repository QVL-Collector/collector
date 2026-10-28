import ArrowBackRounded from '@mui/icons-material/ArrowBackRounded';
import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { LinkButton } from '../../../ui/components/LinkButton';
import { LiveStatus } from '../../../ui/components/LiveStatus';
import { PageHeading } from '../../../ui/components/PageHeading';
import { useArticlePage } from '../hooks/useArticlePage';
import { ArticlePageContent } from './ArticlePageContent';

export function ArticlePage() {
  const { t } = useTranslation();
  const { headingRef, title, subtitle, announcement, ...content } = useArticlePage();
  return (
    <Stack spacing={{ xs: 3, md: 4 }} alignItems="stretch">
      {content.view.status !== 'not-found' && (
        <Stack alignItems="flex-start">
          <LinkButton
            to={content.backPath}
            label={t('common.backToCatalogue')}
            icon={<ArrowBackRounded />}
            variant="text"
          />
        </Stack>
      )}
      <PageHeading headingRef={headingRef} title={title} subtitle={subtitle} />
      <LiveStatus message={announcement} />
      <Stack alignItems={content.view.status === 'not-found' ? 'flex-start' : 'stretch'}>
        <ArticlePageContent {...content} />
      </Stack>
    </Stack>
  );
}
