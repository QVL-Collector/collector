import { useTranslation } from 'react-i18next';
import { LinkButton } from '../../../ui/components/LinkButton';
import type { ArticlePageView } from '../hooks/articlePageView';
import { ArticleDetails } from './ArticleDetails';
import { ArticleDetailsSkeleton } from './ArticleDetailsSkeleton';
import { ArticleErrorState } from './ArticleErrorState';

type ArticlePageContentProps = {
  readonly view: ArticlePageView;
  readonly backPath: string;
  readonly retry: () => void;
};

export function ArticlePageContent({ view, backPath, retry }: ArticlePageContentProps) {
  const { t } = useTranslation();
  switch (view.status) {
    case 'loading':
      return <ArticleDetailsSkeleton />;
    case 'error':
      return <ArticleErrorState onRetry={retry} />;
    case 'not-found':
      return <LinkButton to={backPath} label={t('common.backToCatalogue')} variant="contained" />;
    case 'ready':
      return <ArticleDetails article={view.article} />;
  }
}
