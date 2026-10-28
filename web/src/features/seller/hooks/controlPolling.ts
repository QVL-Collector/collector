import { isUnderControl, type SellerArticle } from '../domain/sellerArticle';

const controlPollingIntervalMs = 2000;

export function pollWhileUnderControl(
  articles: readonly SellerArticle[] | undefined,
): number | false {
  const anyUnderControl = articles?.some((article) => isUnderControl(article.state)) ?? false;
  return anyUnderControl ? controlPollingIntervalMs : false;
}
