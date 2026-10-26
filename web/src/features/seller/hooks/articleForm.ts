import type { FieldProblem } from '../../../api/problemDetails';
import type { ArticleDraft, SellerArticle } from '../domain/sellerArticle';
import { fieldProblemCodes, type FieldProblemCode } from '../domain/sellerProblems';
import { formatAmountInput, parseAmountInput, type AmountInput } from './amountInput';

export const articleFormFields = [
  'title',
  'description',
  'categoryId',
  'price',
  'shippingCost',
] as const;

export type ArticleFormField = (typeof articleFormFields)[number];

export type ArticleFormValues = Readonly<Record<ArticleFormField, string>>;

export type ArticleFieldErrors = Partial<Readonly<Record<ArticleFormField, FieldProblemCode>>>;

export type DraftConversion =
  | { readonly status: 'valid'; readonly draft: ArticleDraft }
  | { readonly status: 'invalid'; readonly errors: ArticleFieldErrors };

export const emptyArticleFormValues: ArticleFormValues = {
  title: '',
  description: '',
  categoryId: '',
  price: '',
  shippingCost: '',
};

const formFieldByPayloadField: Readonly<Record<string, ArticleFormField>> = {
  titre: 'title',
  description: 'description',
  categorieId: 'categoryId',
  prix: 'price',
  fraisDePort: 'shippingCost',
};

export function toArticleFormValues(
  article: SellerArticle,
  decimalSeparator: string,
): ArticleFormValues {
  const amount = (money: SellerArticle['price']) =>
    money === null ? '' : formatAmountInput(money.amountInCents, decimalSeparator);
  return {
    title: article.title,
    description: article.description,
    categoryId: article.category?.id ?? '',
    price: amount(article.price),
    shippingCost: amount(article.shippingCost),
  };
}

function amountInCents(input: AmountInput): number | null {
  return input.status === 'valid' ? input.amountInCents : null;
}

type MutableFieldErrors = Partial<Record<ArticleFormField, FieldProblemCode>>;

function amountErrors(field: ArticleFormField, input: AmountInput): ArticleFieldErrors {
  const errors: MutableFieldErrors = {};
  if (input.status === 'invalid') {
    errors[field] = input.code;
  }
  return errors;
}

export function toArticleDraft(values: ArticleFormValues): DraftConversion {
  const price = parseAmountInput(values.price);
  const shippingCost = parseAmountInput(values.shippingCost);
  const errors = { ...amountErrors('price', price), ...amountErrors('shippingCost', shippingCost) };
  if (Object.keys(errors).length > 0) {
    return { status: 'invalid', errors };
  }
  return {
    status: 'valid',
    draft: {
      title: values.title,
      description: values.description,
      categoryId: values.categoryId === '' ? null : values.categoryId,
      priceInCents: amountInCents(price),
      shippingCostInCents: amountInCents(shippingCost),
    },
  };
}

function toProblemCode(code: string): FieldProblemCode {
  return fieldProblemCodes.find((candidate) => candidate === code) ?? 'invalide';
}

export function toFieldErrors(problems: readonly FieldProblem[]): ArticleFieldErrors {
  const errors: MutableFieldErrors = {};
  problems.forEach((problem) => {
    const field = formFieldByPayloadField[problem.field.split('.')[0] ?? ''];
    if (field !== undefined && errors[field] === undefined) {
      errors[field] = toProblemCode(problem.code);
    }
  });
  return errors;
}

export function withoutFieldError(
  errors: ArticleFieldErrors,
  field: ArticleFormField,
): ArticleFieldErrors {
  const remaining: MutableFieldErrors = { ...errors };
  remaining[field] = undefined;
  return remaining;
}

export function firstInvalidField(errors: ArticleFieldErrors): ArticleFormField | null {
  return articleFormFields.find((field) => errors[field] !== undefined) ?? null;
}

export function countFieldErrors(errors: ArticleFieldErrors): number {
  return articleFormFields.filter((field) => errors[field] !== undefined).length;
}

export function haveSameValues(left: ArticleFormValues, right: ArticleFormValues): boolean {
  return articleFormFields.every((field) => left[field] === right[field]);
}
