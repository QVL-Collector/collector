import { useTranslation } from 'react-i18next';
import { useDateFormatter } from '../../../i18n/useDateFormatter';
import { useMoneyFormatter } from '../../../i18n/useMoneyFormatter';
import type { Money } from '../../../shared/Money';
import type { ReviewArticle } from '../domain/reviewArticle';

export type ReviewFact = {
  readonly label: string;
  readonly value: string;
};

export function useReviewArticleFacts(article: ReviewArticle): readonly ReviewFact[] {
  const { t } = useTranslation();
  const formatMoney = useMoneyFormatter();
  const formatDate = useDateFormatter();
  const notProvided = t('admin.review.notProvided');
  const text = (value: string) => (value.trim() === '' ? notProvided : value);
  const amount = (money: Money | null) => (money === null ? notProvided : formatMoney(money));
  return [
    { label: t('admin.review.facts.description'), value: text(article.description) },
    { label: t('admin.review.facts.category'), value: article.category?.label ?? notProvided },
    { label: t('admin.review.facts.price'), value: amount(article.price) },
    { label: t('admin.review.facts.shippingCost'), value: amount(article.shippingCost) },
    { label: t('admin.review.facts.seller'), value: article.sellerId },
    { label: t('admin.review.facts.updatedAt'), value: formatDate(article.updatedAt) },
  ];
}
