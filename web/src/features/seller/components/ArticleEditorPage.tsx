import Stack from '@mui/material/Stack';
import { LiveStatus } from '../../../ui/components/LiveStatus';
import { PageHeading } from '../../../ui/components/PageHeading';
import { useArticleEditorPage } from '../hooks/useArticleEditorPage';
import { ArticleEditorContent } from './ArticleEditorContent';
import { BackToSellerSpaceLink } from './BackToSellerSpaceLink';

export function ArticleEditorPage() {
  const { headingRef, title, announcement, ...content } = useArticleEditorPage();
  return (
    <Stack spacing={{ xs: 2, md: 3 }}>
      <BackToSellerSpaceLink />
      <PageHeading headingRef={headingRef} title={title} />
      <LiveStatus message={announcement} />
      <ArticleEditorContent {...content} />
    </Stack>
  );
}
