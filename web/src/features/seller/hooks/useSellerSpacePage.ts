import type { RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import { useUserIdentity } from '../../../auth/useUserIdentity';
import { usePageIdentity } from '../../../navigation/usePageIdentity';

export type SellerSpacePageModel = {
  readonly headingRef: RefObject<HTMLHeadingElement | null>;
  readonly title: string;
  readonly welcome: string;
};

export function useSellerSpacePage(): SellerSpacePageModel {
  const { t } = useTranslation();
  const identity = useUserIdentity();
  const headingRef = usePageIdentity(t('seller.documentTitle'));
  return {
    headingRef,
    title: t('seller.title'),
    welcome: t('seller.welcome', { name: identity?.name ?? t('auth.unknownUser') }),
  };
}
