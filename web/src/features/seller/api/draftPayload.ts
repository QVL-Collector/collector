import { articleCurrency, type ArticleDraft } from '../domain/sellerArticle';

export type AmountPayload = {
  readonly montantCentimes: number;
  readonly devise: typeof articleCurrency;
};

export function toAmountPayload(amountInCents: number | null): AmountPayload | null {
  return amountInCents === null
    ? null
    : { montantCentimes: amountInCents, devise: articleCurrency };
}

export function toDraftPayload(draft: ArticleDraft): Readonly<Record<string, unknown>> {
  return {
    titre: draft.title,
    description: draft.description,
    categorieId: draft.categoryId,
    prix: toAmountPayload(draft.priceInCents),
    fraisDePort: toAmountPayload(draft.shippingCostInCents),
  };
}

export function toRevisionPayload(revision: number): Readonly<Record<string, unknown>> {
  return { revision };
}
