import { useTranslation } from 'react-i18next';
import { triggersRecontrol, type ArticleState } from '../domain/sellerArticle';

export type RecontrolWarning = {
  readonly title: string;
  readonly description: string;
};

export function useRecontrolWarning(state: ArticleState | null): RecontrolWarning | null {
  const { t } = useTranslation();
  if (state === null || !triggersRecontrol(state)) {
    return null;
  }
  return state === 'publie'
    ? {
        title: t('seller.form.recontrol.published.title'),
        description: t('seller.form.recontrol.published.description'),
      }
    : {
        title: t('seller.form.recontrol.underReview.title'),
        description: t('seller.form.recontrol.underReview.description'),
      };
}
