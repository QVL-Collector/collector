import type { ControlReason } from '../../../shared/controlReason';
import type { ArticleState } from '../domain/sellerArticle';
import { demoCategoryUnknownToControl, type DemoArticle } from './demoSellerData';

export type DemoControlOutcome = {
  readonly state: Extract<ArticleState, 'publie' | 'refuse' | 'a_verifier'>;
  readonly reason: ControlReason | null;
};

function isComplete(article: DemoArticle): boolean {
  return (
    article.titre.trim() !== '' &&
    article.description.trim() !== '' &&
    article.categorie !== null &&
    article.prix !== null
  );
}

export function controlDemoArticle(article: DemoArticle): DemoControlOutcome {
  if (!isComplete(article)) {
    return { state: 'refuse', reason: 'annonce_incomplete' };
  }
  if (article.categorie?.id === demoCategoryUnknownToControl) {
    return { state: 'a_verifier', reason: 'categorie_inconnue' };
  }
  return { state: 'publie', reason: null };
}
