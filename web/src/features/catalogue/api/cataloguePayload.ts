import { parseCategory, parseMoney, parseNullable } from '../../../shared/marketplacePayload';
import {
  expectArray,
  expectInteger,
  expectNonEmptyString,
  expectRecord,
  expectString,
} from '../../../shared/payload';
import type { CatalogueArticle, CataloguePage, PublishedArticle } from '../domain/catalogue';

function parseArticle(value: unknown, path: string): CatalogueArticle {
  const article = expectRecord(value, path);
  return {
    id: expectNonEmptyString(article.id, `${path}.id`),
    title: expectNonEmptyString(article.titre, `${path}.titre`),
    price: parseMoney(article.prix, `${path}.prix`),
    category: parseCategory(article.categorie, `${path}.categorie`),
  };
}

export function parseCataloguePage(payload: unknown): CataloguePage {
  const root = expectRecord(payload, 'catalogue');
  return {
    articles: expectArray(root.elements, 'elements').map((item, index) =>
      parseArticle(item, `elements[${String(index)}]`),
    ),
    page: expectInteger(root.page, 'page', 1),
    pageSize: expectInteger(root.taille, 'taille', 1),
    totalItems: expectInteger(root.total, 'total', 0),
  };
}

export function parsePublishedArticle(payload: unknown): PublishedArticle {
  const path = 'article';
  const article = expectRecord(payload, path);
  return {
    ...parseArticle(article, path),
    description: expectString(article.description, `${path}.description`),
    shippingCost: parseNullable(article.fraisDePort, (shipping) =>
      parseMoney(shipping, `${path}.fraisDePort`),
    ),
  };
}
