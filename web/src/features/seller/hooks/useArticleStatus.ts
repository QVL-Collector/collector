import { isUnderControl, type ArticleState, type SellerArticle } from '../domain/sellerArticle';
import { useArticleLabels } from './useArticleLabels';

export type ArticleStatus = {
  readonly state: ArticleState;
  readonly label: string;
  readonly detail: string;
  readonly reason: string | null;
  readonly isUnderControl: boolean;
};

export function useArticleStatus(article: SellerArticle): ArticleStatus {
  const { stateLabel, stateDetail, reasonOf } = useArticleLabels();
  return {
    state: article.state,
    label: stateLabel(article.state),
    detail: stateDetail(article.state),
    reason: reasonOf(article),
    isUnderControl: isUnderControl(article.state),
  };
}
