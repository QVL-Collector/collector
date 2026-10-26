import { useTranslation } from 'react-i18next';
import { useMoneyFormatter } from '../../../i18n/useMoneyFormatter';
import { CardGrid } from '../../../ui/components/CardGrid';
import { CardGridItem } from '../../../ui/components/CardGridItem';
import type { CatalogueArticle } from '../domain/catalogue';
import { ArticleCard } from './ArticleCard';

type CatalogueGridProps = {
  readonly articles: readonly CatalogueArticle[];
};

export function CatalogueGrid({ articles }: CatalogueGridProps) {
  const { t } = useTranslation();
  const formatMoney = useMoneyFormatter();
  return (
    <CardGrid label={t('catalogue.listLabel')}>
      {articles.map((article, index) => (
        <CardGridItem key={article.id} index={index}>
          <ArticleCard article={article} priceLabel={formatMoney(article.price)} />
        </CardGridItem>
      ))}
    </CardGrid>
  );
}
