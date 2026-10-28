import { useTranslation } from 'react-i18next';
import { useMoneyFormatter } from '../../../i18n/useMoneyFormatter';
import type { Money } from '../../../shared/Money';
import type { SellerArticle } from '../domain/sellerArticle';

export type SummaryEntry = {
  readonly label: string;
  readonly value: string;
};

export function useArticleSummary(article: SellerArticle): readonly SummaryEntry[] {
  const { t } = useTranslation();
  const formatMoney = useMoneyFormatter();
  const notProvided = t('seller.notProvided');
  const text = (value: string) => (value.trim() === '' ? notProvided : value);
  const amount = (money: Money | null) => (money === null ? notProvided : formatMoney(money));
  return [
    { label: t('seller.form.title'), value: text(article.title) },
    { label: t('seller.form.description'), value: text(article.description) },
    { label: t('seller.form.category'), value: article.category?.label ?? notProvided },
    { label: t('seller.form.price'), value: amount(article.price) },
    { label: t('seller.form.shippingCost'), value: amount(article.shippingCost) },
  ];
}
