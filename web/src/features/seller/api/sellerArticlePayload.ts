import { controlReasons, type ControlReason } from '../../../shared/controlReason';
import { parseCategory, parseMoney, parseNullable } from '../../../shared/marketplacePayload';
import {
  expectArray,
  expectInteger,
  expectNonEmptyString,
  expectOneOf,
  expectRecord,
  expectString,
  InvalidPayloadError,
} from '../../../shared/payload';
import {
  articleStates,
  type SellerArticle,
  type SellerArticlePage,
  type SellerCategory,
} from '../domain/sellerArticle';

function parseReason(value: unknown, path: string): ControlReason | null {
  return parseNullable(value, (reason) => expectOneOf(reason, controlReasons, path));
}

function parseDate(value: unknown, path: string): Date {
  const date = new Date(expectNonEmptyString(value, path));
  if (Number.isNaN(date.getTime())) {
    throw new InvalidPayloadError(path);
  }
  return date;
}

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function parseUuid(value: unknown, path: string): string {
  const id = expectString(value, path);
  if (!uuidPattern.test(id)) {
    throw new InvalidPayloadError(path);
  }
  return id;
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

export function parseSellerCategories(payload: unknown): readonly SellerCategory[] {
  const root = expectRecord(payload, 'categories');
  return expectArray(root.elements, 'elements').map((item, index) =>
    parseCategory(item, `elements[${String(index)}]`),
  );
}
