import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { buyer, press } from '../US-008/identityProviderHarness';
import {
  amountShownFor,
  articleId,
  deadline,
  failureTimeout,
  formatted,
  givenArticles,
  givenExistingOrder,
  givenNextOrder,
  openArticleAs,
  openOrderAs,
  order,
  orderId,
  orderReadAnswers,
  paymentDeadline,
  publishedArticle,
  wording,
} from './reservationHarness';

const languages: readonly SupportedLanguage[] = ['fr', 'en'];

type Amounts = { prix: number; frais: number; total: number; commission: number; vendeur: number };

const standardAmounts: Amounts = {
  prix: 12000,
  frais: 590,
  total: 12590,
  commission: 630,
  vendeur: 11960,
};
const freeShipping: Amounts = { prix: 4500, frais: 0, total: 4500, commission: 225, vendeur: 4275 };
const halfCent: Amounts = { prix: 10, frais: 0, total: 10, commission: 1, vendeur: 9 };
const largeAmounts: Amounts = {
  prix: 100000000,
  frais: 100000,
  total: 100100000,
  commission: 5005000,
  vendeur: 95095000,
};

function expectAmounts(amounts: Amounts, language: SupportedLanguage): void {
  const words = wording[language];
  expect(amountShownFor(words.price)).toBe(formatted(amounts.prix, language));
  expect(amountShownFor(words.shipping)).toBe(formatted(amounts.frais, language));
  expect(amountShownFor(words.total)).toBe(formatted(amounts.total, language));
  expect(amountShownFor(words.commission)).toBe(formatted(amounts.commission, language));
  expect(amountShownFor(words.sellerAmount)).toBe(formatted(amounts.vendeur, language));
}

async function openConfirmationOf(amounts: Amounts, language: SupportedLanguage): Promise<void> {
  givenExistingOrder(order(1, articleId(1), amounts));
  await openOrderAs(orderId(1), language);
  await screen.findByRole('heading', { level: 1, name: wording[language].title }, failureTimeout);
}

describe('US-030-AC2 : écran de confirmation de la commande', () => {
  it.each(languages)(
    'Given la langue %s et une commande créée, When l’acheteur la consulte, Then il voit prix, frais de port, total, commission de 5 % et montant vendeur au centime',
    async (language) => {
      await openConfirmationOf(standardAmounts, language);

      expectAmounts(standardAmounts, language);
    },
  );

  it.each(languages)(
    'Given la langue %s et une commande créée, When l’acheteur la consulte, Then l’échéance fixée par la Place de marché est affichée',
    async (language) => {
      await openConfirmationOf(standardAmounts, language);

      expect(paymentDeadline(language).dateTime).toBe(new Date(deadline).toISOString());
      expect(paymentDeadline(language).textContent).toMatch(/2026/);
    },
  );

  it('Given un article sans frais de port, When l’acheteur consulte la commande, Then les frais valent 0 et la commission porte sur le seul prix', async () => {
    await openConfirmationOf(freeShipping, 'fr');

    expectAmounts(freeShipping, 'fr');
  });

  it('Given un total d’un demi-centime de commission, When l’acheteur consulte la commande, Then les montants sont affichés au centime tels que calculés par le serveur', async () => {
    await openConfirmationOf(halfCent, 'fr');

    expectAmounts(halfCent, 'fr');
  });

  it('Given un grand montant, When l’acheteur consulte la commande, Then les montants sont affichés au centime sans perte', async () => {
    await openConfirmationOf(largeAmounts, 'en');

    expectAmounts(largeAmounts, 'en');
  });

  it('Given un acheteur qui vient de réserver, When l’écran de confirmation s’affiche, Then il porte les montants et l’échéance renvoyés par Transactions', async () => {
    givenArticles(publishedArticle(1));
    givenNextOrder(order(1, articleId(1), standardAmounts));
    await openArticleAs(buyer, articleId(1));

    await press(await screen.findByRole('button', { name: wording.fr.reserve }, failureTimeout));

    await screen.findByRole('heading', { level: 1, name: wording.fr.title }, failureTimeout);
    expectAmounts(standardAmounts, 'fr');
    expect(paymentDeadline('fr').dateTime).toBe(new Date(deadline).toISOString());
  });

  it.each(languages)(
    'Given la langue %s et une commande inconnue ou d’un autre acheteur (404), When l’acheteur ouvre son adresse, Then « commande introuvable » s’affiche sans montant',
    async (language) => {
      orderReadAnswers(404);
      await openOrderAs(orderId(9), language);

      expect(
        await screen.findByRole(
          'heading',
          { level: 1, name: wording[language].notFound },
          failureTimeout,
        ),
      ).toBeInTheDocument();
      expect(screen.queryByText(wording[language].total)).not.toBeInTheDocument();
    },
  );
});
