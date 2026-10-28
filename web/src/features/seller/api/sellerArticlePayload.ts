import { articleReasons, type ArticleReason } from '../../../shared/articleReason';
import {
  parseCategory,
  parseDate,
  parseMoney,
  parseNullable,
  parseUuid,
} from '../../../shared/marketplacePayload';
import {
  expectArray,
  expectInteger,
  expectOneOf,
  expectRecord,
  expectString,
} from '../../../shared/payload';
import { articleStates, type SellerArticle, type SellerArticlePage } from '../domain/sellerArticle';

function parseReason(value: unknown, path: string): ArticleReason | null {
  return parseNullable(value, (reason) => expectOneOf(reason, articleReasons, path));
}

export function parseSellerArticle(value: unknown, path = 'article'): SellerArticle {
  const article = expectRecord(value, path);
  parseDate(article.creeLe, `${path}.creeLe`);
  return {
    id: parseUuid(article.id, `${path}.id`),
    state: expectOneOf(article.etat, articleStates, `${path}.etat`),
    revision: expectInteger(article.revision, `${path}.revision`, 1),
    title: expectString(article.titre, `${path}.titre`),
    description: expectString(article.description, `${path}.description`),
    category: parseNullable(article.categorie, (category) =>
      parseCategory(category, `${path}.categorie`),
    ),
    price: parseNullable(article.prix, (price) => parseMoney(price, `${path}.prix`)),
    shippingCost: parseNullable(article.fraisDePort, (shipping) =>
      parseMoney(shipping, `${path}.fraisDePort`),
    ),
    reason: parseReason(article.motif, `${path}.motif`),
    updatedAt: parseDate(article.modifieLe, `${path}.modifieLe`),
  };
}

export function parseSellerArticlePage(payload: unknown): SellerArticlePage {
  const root = expectRecord(payload, 'articles');
  return {
    articles: expectArray(root.elements, 'elements').map((item, index) =>
      parseSellerArticle(item, `elements[${String(index)}]`),
    ),
    page: expectInteger(root.page, 'page', 1),
    pageSize: expectInteger(root.taille, 'taille', 1),
    totalItems: expectInteger(root.total, 'total', 0),
  };
}
