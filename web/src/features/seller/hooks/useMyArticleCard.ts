import { useTranslation } from 'react-i18next';
import { useDateFormatter } from '../../../i18n/useDateFormatter';
import { useMoneyFormatter } from '../../../i18n/useMoneyFormatter';
import type { SellerArticle } from '../domain/sellerArticle';
import { articlePath } from '../sellerPaths';
import { useArticleStatus, type ArticleStatus } from './useArticleStatus';
import { useArticleLabels } from './useArticleLabels';

export type MyArticleCardModel = {
  readonly path: string;
  readonly title: string;
  readonly priceLabel: string;
  readonly updatedLabel: string;
  readonly status: ArticleStatus;
};

export function useMyArticleCard(article: SellerArticle): MyArticleCardModel {
  const { t } = useTranslation();
  const formatMoney = useMoneyFormatter();
  const formatDate = useDateFormatter();
  const { titleOf } = useArticleLabels();
  const status = useArticleStatus(article);
  return {
    path: articlePath(article.id),
    title: titleOf(article),
    priceLabel: article.price === null ? t('seller.list.noPrice') : formatMoney(article.price),
    updatedLabel: t('seller.list.updatedAt', { date: formatDate(article.updatedAt) }),
    status,
  };
}
