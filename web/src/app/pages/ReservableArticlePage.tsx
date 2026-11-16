import { ArticlePage } from '../../features/catalogue/components/ArticlePage';
import type { PublishedArticle } from '../../features/catalogue/domain/catalogue';
import { ArticleReservation } from '../../features/orders/components/ArticleReservation';

function reservationOf(article: PublishedArticle) {
  return <ArticleReservation key={article.id} articleId={article.id} reserved={article.reserved} />;
}

export function ReservableArticlePage() {
  return <ArticlePage purchaseAction={reservationOf} />;
}
