import RefreshRounded from '@mui/icons-material/RefreshRounded';
import SearchOffRounded from '@mui/icons-material/SearchOffRounded';
import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';
import { EmptyState } from '../../../ui/components/EmptyState';
import { ErrorState } from '../../../ui/components/ErrorState';
import { PendingState } from '../../../ui/components/PendingState';
import type { SellerArticle } from '../domain/sellerArticle';
import type { ArticleEditorPageView } from '../hooks/useArticleEditorPage';
import { ArticleWorkspace } from './ArticleWorkspace';

type ArticleEditorContentProps = {
  readonly view: ArticleEditorPageView;
  readonly retry: () => void;
  readonly fetchLatest: () => Promise<SellerArticle | null>;
};

export function ArticleEditorContent({ view, retry, fetchLatest }: ArticleEditorContentProps) {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
      return <PendingState message={t('seller.editor.loading')} />;
    case 'not-found':
      return (
        <EmptyState
          icon={<SearchOffRounded fontSize="inherit" />}
          title={t('seller.editor.notFound.title')}
          description={t('seller.editor.notFound.description')}
        />
      );
    case 'error':
      return (
        <ErrorState
          title={t('seller.editor.error.title')}
          description={t('seller.editor.error.description')}
          action={
            <Button variant="outlined" color="error" startIcon={<RefreshRounded />} onClick={retry}>
              {t('seller.editor.error.retry')}
            </Button>
          }
        />
      );
    case 'ready':
      return (
        <ArticleWorkspace key={view.article.id} article={view.article} fetchLatest={fetchLatest} />
      );
  }
}
