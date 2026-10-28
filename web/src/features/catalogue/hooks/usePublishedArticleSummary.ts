import { useTranslation } from 'react-i18next';
import { useMoneyFormatter } from '../../../i18n/useMoneyFormatter';
import { categoryCataloguePath } from '../cataloguePaths';
import type { PublishedArticle } from '../domain/catalogue';

export type PublishedArticleSummary = {
  readonly priceLabel: string;
  readonly shippingLabel: string;
  readonly categoryLabel: string;
  readonly categoryPath: string;
  readonly description: string | null;
};

export function usePublishedArticleSummary(article: PublishedArticle): PublishedArticleSummary {
  const { t } = useTranslation();
  const formatMoney = useMoneyFormatter();
  const description = article.description.trim();
  return {
    priceLabel: formatMoney(article.price),
    shippingLabel:
      article.shippingCost === null
        ? t('article.shippingNotProvided')
        : formatMoney(article.shippingCost),
    categoryLabel: article.category.label,
    categoryPath: categoryCataloguePath(article.category.id),
    description: description === '' ? null : article.description,
  };
}
