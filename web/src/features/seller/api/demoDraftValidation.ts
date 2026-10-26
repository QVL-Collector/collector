import type { FieldProblem } from '../../../api/problemDetails';
import { articleLimits, type ArticleDraft } from '../domain/sellerArticle';
import type { FieldProblemCode } from '../domain/sellerProblems';
import { demoCategories } from './demoSellerData';

function problem(field: string, code: FieldProblemCode): readonly FieldProblem[] {
  return [{ field, code }];
}

function textProblems(field: string, text: string, maxLength: number): readonly FieldProblem[] {
  return text.length > maxLength ? problem(field, 'trop_long') : [];
}

function amountProblems(
  field: string,
  amountInCents: number | null,
  maxInCents: number,
): readonly FieldProblem[] {
  if (amountInCents === null) {
    return [];
  }
  const amountField = `${field}.montantCentimes`;
  if (amountInCents < 0) {
    return problem(amountField, 'negatif');
  }
  return amountInCents > maxInCents ? problem(amountField, 'au_dela_du_plafond') : [];
}

function categoryProblems(categoryId: string | null): readonly FieldProblem[] {
  const known = categoryId === null || demoCategories.some(({ id }) => id === categoryId);
  return known ? [] : problem('categorieId', 'categorie_inconnue');
}

export function validateDemoDraft(draft: ArticleDraft): readonly FieldProblem[] {
  const { maxPriceInCents, maxShippingCostInCents } = articleLimits;
  return [
    ...textProblems('titre', draft.title, articleLimits.titleMaxLength),
    ...textProblems('description', draft.description, articleLimits.descriptionMaxLength),
    ...categoryProblems(draft.categoryId),
    ...amountProblems('prix', draft.priceInCents, maxPriceInCents),
    ...amountProblems('fraisDePort', draft.shippingCostInCents, maxShippingCostInCents),
  ];
}
