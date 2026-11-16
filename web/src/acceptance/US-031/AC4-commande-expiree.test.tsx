import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { axeTimeout, blockingViolations } from '../US-013/sellerMarketplaceHarness';
import {
  amountShownFor,
  articleId,
  deadline,
  failureTimeout,
  formatted,
  givenExistingOrder,
  openOrderAs,
  order,
  orderId,
  wording,
  type OrderWire,
} from '../US-030/reservationHarness';

const languages: readonly SupportedLanguage[] = ['fr', 'en'];

type ExpiredWording = {
  readonly title: string;
  readonly status: string;
  readonly message: string;
  readonly deadlinePassed: string;
};

const expiredWording: Readonly<Record<SupportedLanguage, ExpiredWording>> = {
  fr: {
    title: 'Réservation expirée',
    status: 'Expirée',
    message: 'La réservation a expiré, l’article est de nouveau disponible.',
    deadlinePassed: 'Échéance dépassée le',
  },
  en: {
    title: 'Reservation expired',
    status: 'Expired',
    message: 'The reservation has expired, the item is available again.',
    deadlinePassed: 'Deadline passed on',
  },
};

const amounts = { prix: 8000, frais: 500, total: 8500, commission: 425, vendeur: 8075 };

function expiredOrder(): OrderWire {
  return { ...order(1, articleId(7), amounts), statut: 'expiree' };
}

async function openExpiredOrder(language: SupportedLanguage): Promise<void> {
  givenExistingOrder(expiredOrder());
  await openOrderAs(orderId(1), language);
  await screen.findByRole(
    'heading',
    { level: 1, name: expiredWording[language].title },
    failureTimeout,
  );
}

function linksToArticle(id: string): HTMLElement[] {
  return screen
    .queryAllByRole('link')
    .filter((link) => (link.getAttribute('href') ?? '').endsWith(`/articles/${id}`));
}

describe('US-031-AC4 : commande présentée « expirée » après l’échéance', () => {
  it.each(languages)(
    'Given la langue %s et une commande au statut expiree, When l’acheteur la consulte, Then le titre et le statut disent « expirée »',
    async (language) => {
      await openExpiredOrder(language);

      expect(screen.getAllByText(expiredWording[language].status).length).toBeGreaterThan(0);
      expect(
        screen.queryByRole('heading', { level: 1, name: wording[language].title }),
      ).not.toBeInTheDocument();
      expect(screen.queryByText(wording[language].status)).not.toBeInTheDocument();
    },
  );

  it.each(languages)(
    'Given la langue %s et une commande expirée, When l’acheteur la consulte, Then un message dit que l’article est de nouveau disponible',
    async (language) => {
      await openExpiredOrder(language);

      expect(screen.getByText(expiredWording[language].message)).toBeInTheDocument();
    },
  );

  it.each(languages)(
    'Given la langue %s et une commande expirée, When l’acheteur la consulte, Then un lien mène à la fiche de l’article',
    async (language) => {
      await openExpiredOrder(language);

      expect(linksToArticle(articleId(7)).length).toBeGreaterThan(0);
    },
  );

  it.each(languages)(
    'Given la langue %s et une commande expirée, When l’acheteur la consulte, Then l’échéance est présentée comme dépassée et non plus à régler',
    async (language) => {
      await openExpiredOrder(language);

      const caption = screen.getByText(expiredWording[language].deadlinePassed);
      const time = caption.parentElement?.querySelector('time');
      expect(time?.dateTime).toBe(new Date(deadline).toISOString());
      expect(screen.queryByText(wording[language].paymentDue)).not.toBeInTheDocument();
    },
  );

  it.each(languages)(
    'Given la langue %s et une commande expirée, When l’acheteur la consulte, Then elle reste lisible avec ses montants',
    async (language) => {
      await openExpiredOrder(language);

      expect(amountShownFor(wording[language].price)).toBe(formatted(amounts.prix, language));
      expect(amountShownFor(wording[language].total)).toBe(formatted(amounts.total, language));
    },
  );

  it('Given une commande expirée, When l’acheteur la consulte, Then aucune action de paiement ni de réservation n’est proposée sur cet écran', async () => {
    await openExpiredOrder('fr');

    const main = screen.getByRole('main');
    expect(
      within(main).queryByRole('button', { name: wording.fr.reserve }),
    ).not.toBeInTheDocument();
    expect(within(main).queryByRole('button', { name: /payer/i })).not.toBeInTheDocument();
  });

  it('Given une commande encore en attente de paiement, When l’acheteur la consulte, Then rien n’indique une expiration', async () => {
    givenExistingOrder(order(1, articleId(7), amounts));
    await openOrderAs(orderId(1), 'fr');
    await screen.findByRole('heading', { level: 1, name: wording.fr.title }, failureTimeout);

    expect(screen.getAllByText(wording.fr.status).length).toBeGreaterThan(0);
    expect(screen.queryByText(expiredWording.fr.status)).not.toBeInTheDocument();
    expect(screen.queryByText(expiredWording.fr.message)).not.toBeInTheDocument();
  });

  it.each(languages)(
    'Given la langue %s et l’écran d’une commande expirée, When axe-core l’analyse, Then aucune violation serious ou critical',
    async (language) => {
      await openExpiredOrder(language);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );
});
