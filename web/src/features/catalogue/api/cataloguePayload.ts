import { isCurrencyCode, type Money } from '../../../shared/Money';
import {
  expectArray,
  expectInteger,
  expectNonEmptyString,
  expectRecord,
  InvalidPayloadError,
} from '../../../shared/payload';
import type { ArticleCategory, CatalogueArticle, CataloguePage } from '../domain/catalogue';

function parseMoney(value: unknown, path: string): Money {
  const price = expectRecord(value, path);
  const currency = expectNonEmptyString(price.devise, `${path}.devise`);
  if (!isCurrencyCode(currency)) {
    throw new InvalidPayloadError(`${path}.devise`);
  }
  return {
    amountInCents: expectInteger(price.montantCentimes, `${path}.montantCentimes`, 0),
    currency,
  };
}

function parseCategory(value: unknown, path: string): ArticleCategory {
  const category = expectRecord(value, path);
  return {
    id: expectNonEmptyString(category.id, `${path}.id`),
    label: expectNonEmptyString(category.libelle, `${path}.libelle`),
  };
}

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
