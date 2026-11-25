import Stack from '@mui/material/Stack';
import { SellerPhotoSection } from '../../photos/components/SellerPhotoSection';
import type { SellerArticle } from '../domain/sellerArticle';
import { useArticleWorkspace } from '../hooks/useArticleWorkspace';
import { ArticleForm } from './ArticleForm';
import { ArticleStatusPanel } from './ArticleStatusPanel';
import { ArticleSubmitAction } from './ArticleSubmitAction';

type ArticleWorkspaceProps = {
  readonly article: SellerArticle;
  readonly fetchLatest: () => Promise<SellerArticle | null>;
};

export function ArticleWorkspace({ article, fetchLatest }: ArticleWorkspaceProps) {
  const { status, editor, submission, categories, lockMessage, isSubmissionBlocked, photos } =
    useArticleWorkspace(article, fetchLatest);
  return (
    <Stack
      direction={{ xs: 'column', md: 'row-reverse' }}
      spacing={{ xs: 3, md: 5 }}
      alignItems={{ xs: 'stretch', md: 'flex-start' }}
      useFlexGap
    >
      <Stack width={{ xs: '100%', md: '22rem' }} flexShrink={0}>
        <ArticleStatusPanel
          status={status}
          notice={lockMessage}
          action={
            submission.canSubmit ? (
              <ArticleSubmitAction
                isSubmitting={submission.isSubmitting}
                isBlocked={isSubmissionBlocked}
                failure={submission.failure}
                onSubmit={submission.submit}
              />
            ) : undefined
          }
        />
      </Stack>
      <Stack flexGrow={1} minWidth={0} spacing={{ xs: 3, md: 4 }}>
        <ArticleForm editor={editor} categories={categories} />
        <SellerPhotoSection {...photos} />
      </Stack>
    </Stack>
  );
}
