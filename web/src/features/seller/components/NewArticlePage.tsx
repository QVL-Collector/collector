import Stack from '@mui/material/Stack';
import { PageHeading } from '../../../ui/components/PageHeading';
import { useNewArticlePage } from '../hooks/useNewArticlePage';
import { ArticleForm } from './ArticleForm';
import { BackToSellerSpaceLink } from './BackToSellerSpaceLink';

export function NewArticlePage() {
  const { headingRef, title, subtitle, editor, categories } = useNewArticlePage();
  return (
    <Stack spacing={{ xs: 2, md: 3 }}>
      <BackToSellerSpaceLink />
      <PageHeading headingRef={headingRef} title={title} subtitle={subtitle} />
      <Stack maxWidth="48rem" paddingTop={{ xs: 1, md: 2 }}>
        <ArticleForm editor={editor} categories={categories} />
      </Stack>
    </Stack>
  );
}
