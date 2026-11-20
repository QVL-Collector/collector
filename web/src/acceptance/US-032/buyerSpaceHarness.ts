import { screen, within } from '@testing-library/react';
import { expect } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { buyer, signInAs, type FictitiousAccount } from '../US-008/identityProviderHarness';
import {
  articleId,
  backends,
  failureTimeout,
  order,
  type OrderWire,
} from '../US-030/reservationHarness';

export const buyerSpacePath = '/acheteur';

export const languages: readonly SupportedLanguage[] = ['fr', 'en'];

export type BuyerSpaceWording = {
  readonly navigation: string;
  readonly title: string;
  readonly listLabel: string;
  readonly awaitingPayment: string;
  readonly expired: string;
  readonly emptyTitle: string;
  readonly emptyAction: string;
  readonly errorTitle: string;
  readonly retry: string;
  readonly viewDetail: (title: string) => string;
  readonly accessDenied: string;
  readonly signIn: string;
};

export const buyerSpaceWording: Readonly<Record<SupportedLanguage, BuyerSpaceWording>> = {
  fr: {
    navigation: 'Mes réservations',
    title: 'Mes réservations',
    listLabel: 'Réservations',
    awaitingPayment: 'En attente de paiement',
    expired: 'Expirée',
    emptyTitle: 'Aucune réservation pour le moment',
    emptyAction: 'Parcourir le catalogue',
    errorTitle: 'Vos réservations sont momentanément indisponibles',
    retry: 'Réessayer',
    viewDetail: (title) => `Voir la réservation « ${title} »`,
    accessDenied: 'Accès réservé',
    signIn: 'Se connecter',
  },
  en: {
    navigation: 'My reservations',
    title: 'My reservations',
    listLabel: 'Reservations',
    awaitingPayment: 'Awaiting payment',
    expired: 'Expired',
    emptyTitle: 'No reservations yet',
    emptyAction: 'Browse the catalogue',
    errorTitle: 'Your reservations are temporarily unavailable',
    retry: 'Try again',
    viewDetail: (title) => `View the reservation “${title}”`,
    accessDenied: 'Restricted access',
    signIn: 'Sign in',
  },
};

export const nominalAmounts = {
  prix: 12000,
  frais: 590,
  total: 12590,
  commission: 630,
  vendeur: 11960,
};

export function inHours(hours: number): string {
  return new Date(Date.now() + hours * 3_600_000).toISOString();
}

export function pendingOrder(
  rank: number,
  title: string,
  deadline: string = inHours(2),
): OrderWire {
  return { ...order(rank, articleId(rank), nominalAmounts), titre: title, echeance: deadline };
}

export function expiredOrder(rank: number, title: string): OrderWire {
  return { ...pendingOrder(rank, title, inHours(-1)), statut: 'expiree' };
}

export function givenBuyerOrders(...orders: OrderWire[]): void {
  backends().orders = orders;
}

export function replaceOrder(updated: OrderWire): void {
  backends().orders = backends().orders.map((existing) =>
    existing.id === updated.id ? updated : existing,
  );
}

export async function openBuyerSpace(
  language: SupportedLanguage = 'fr',
  account: FictitiousAccount = buyer,
): Promise<void> {
  await signInAs(account, buyerSpacePath, language);
  await screen.findByRole(
    'heading',
    { level: 1, name: buyerSpaceWording[language].title },
    failureTimeout,
  );
}

export async function reservationList(language: SupportedLanguage): Promise<HTMLElement> {
  return screen.findByRole('list', { name: buyerSpaceWording[language].listLabel }, failureTimeout);
}

export async function reservationItems(language: SupportedLanguage): Promise<HTMLElement[]> {
  return within(await reservationList(language)).findAllByRole('listitem', {}, failureTimeout);
}

export function itemFor(items: readonly HTMLElement[], title: string): HTMLElement {
  const found = items.find((item) => within(item).queryByText(title) !== null);
  if (found === undefined) {
    throw new Error(`Réservation « ${title} » absente de la liste`);
  }
  return found;
}

export function deadlinesIn(item: HTMLElement): number[] {
  return Array.from(item.querySelectorAll('time')).map((time) => new Date(time.dateTime).getTime());
}

export function normalizedText(element: HTMLElement): string {
  return element.textContent.replace(/\s+/g, ' ');
}

export function expectState(item: HTMLElement, expected: string, absent: string): void {
  expect(within(item).getAllByText(expected).length).toBeGreaterThan(0);
  expect(within(item).queryByText(absent)).not.toBeInTheDocument();
}
