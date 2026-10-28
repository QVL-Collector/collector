import { useCallback, useState, type RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { useUserIdentity } from '../../../auth/useUserIdentity';
import { usePageIdentity } from '../../../navigation/usePageIdentity';
import { toMyArticlesView, type MyArticlesView } from './myArticlesView';
import { useMyArticles } from './useMyArticles';
import { useMyArticlesAnnouncement } from './useMyArticlesAnnouncement';
import { useStateChangeAnnouncement } from './useStateChangeAnnouncement';

export type SellerSpacePageModel = {
  readonly headingRef: RefObject<HTMLHeadingElement | null>;
  readonly title: string;
  readonly welcome: string;
  readonly view: MyArticlesView;
  readonly listAnnouncement: string;
  readonly stateAnnouncement: string;
  readonly retry: () => void;
  readonly goToPage: (page: number) => void;
};

export function useSellerSpacePage(): SellerSpacePageModel {
  const { t } = useTranslation();
  const identity = useUserIdentity();
  const headingRef = usePageIdentity(t('seller.documentTitle'));
  const [page, setPage] = useState(1);
  const query = useMyArticles(page);
  const view = toMyArticlesView(query, page);
  const listAnnouncement = useMyArticlesAnnouncement(view);
  const stateAnnouncement = useStateChangeAnnouncement(query.data?.articles);
  const { refetch } = query;
  const retry = useCallback(() => {
    void refetch();
  }, [refetch]);
  return {
    headingRef,
    title: t('seller.title'),
    welcome: t('seller.welcome', { name: identity?.name ?? t('auth.unknownUser') }),
    view,
    listAnnouncement,
    stateAnnouncement,
    retry,
    goToPage: setPage,
  };
}
