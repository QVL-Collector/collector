import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { buyer, openCollector } from '../US-008/identityProviderHarness';
import { axeTimeout, blockingViolations } from '../US-013/sellerMarketplaceHarness';
import {
  articleId,
  failureTimeout,
  givenArticles,
  givenExistingOrder,
  openArticleAs,
  openOrderAs,
  order,
  orderId,
  publishedArticle,
  wording,
} from './reservationHarness';

const languages: readonly SupportedLanguage[] = ['fr', 'en'];

type Situation = {
  readonly name: string;
  readonly open: (language: SupportedLanguage) => Promise<void>;
};

const situations: readonly Situation[] = [
  {
    name: 'fiche réservable par un acheteur',
    open: async (language) => {
      givenArticles(publishedArticle(1));
      await openArticleAs(buyer, articleId(1), language);
      await screen.findByRole('button', { name: wording[language].reserve }, failureTimeout);
    },
  },
  {
    name: 'fiche d’un article réservé',
    open: async (language) => {
      givenArticles(publishedArticle(1, { reserve: true }));
      await openArticleAs(null, articleId(1), language);
      await screen.findByText(wording[language].reservedReason, {}, failureTimeout);
    },
  },
  {
    name: 'catalogue avec un article réservé',
    open: async (language) => {
      givenArticles(publishedArticle(1, { reserve: true }), publishedArticle(2));
      await openCollector('/', language);
      await screen.findByText(wording[language].reserved, {}, failureTimeout);
    },
  },
  {
    name: 'écran de confirmation',
    open: async (language) => {
      givenExistingOrder(
        order(1, articleId(1), {
          prix: 12000,
          frais: 590,
          total: 12590,
          commission: 630,
          vendeur: 11960,
        }),
      );
      await openOrderAs(orderId(1), language);
      await screen.findByRole(
        'heading',
        { level: 1, name: wording[language].title },
        failureTimeout,
      );
    },
  },
];

const cases = situations.flatMap((situation) =>
  languages.map((language) => ({ ...situation, language })),
);

describe('US-030 accessibilité des écrans de réservation (axe-core, jsdom)', () => {
  it.each(cases)(
    'Given l’écran « $name » en $language, When axe-core l’analyse, Then aucune violation sérieuse ou critique',
    async ({ open, language }) => {
      await open(language);

      expect(await blockingViolations()).toStrictEqual([]);
    },
    axeTimeout,
  );
});
