import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import type { SupportedLanguage } from '../../i18n/languages';
import { buyer, currentPath, press } from '../US-008/identityProviderHarness';
import {
  articleId,
  articlePath,
  failureTimeout,
  givenArticles,
  openArticleAs,
  publishedArticle,
  reservationAnswers,
  waitForReservationRequest,
  wording,
  type ReservationWording,
} from './reservationHarness';

const languages: readonly SupportedLanguage[] = ['fr', 'en'];

type Refusal = {
  readonly name: string;
  readonly status: number;
  readonly type?: string;
  readonly message: (words: ReservationWording) => string;
};

const refusals: readonly Refusal[] = [
  {
    name: '422 article non réservable',
    status: 422,
    type: 'urn:collector:probleme:article-non-reservable',
    message: (words) => words.failures.notReservable,
  },
  {
    name: '409 article déjà réservé',
    status: 409,
    type: 'urn:collector:probleme:article-deja-reserve',
    message: (words) => words.failures.alreadyReserved,
  },
  {
    name: '409 réservation déjà active',
    status: 409,
    type: 'urn:collector:probleme:reservation-deja-active',
    message: (words) => words.failures.reservationAlreadyActive,
  },
  { name: '401 session expirée', status: 401, message: (words) => words.failures.sessionExpired },
  { name: '403 sans rôle acheteur', status: 403, message: (words) => words.failures.forbidden },
  { name: '429 trop de tentatives', status: 429, message: (words) => words.failures.rateLimited },
  {
    name: '503 Place de marché indisponible',
    status: 503,
    message: (words) => words.failures.unavailable,
  },
];

const cases = refusals.flatMap((refusal) =>
  languages.map((language) => ({ ...refusal, language })),
);

describe('US-030-AC1 : réservation refusée par Transactions', () => {
  it.each(cases)(
    'Given la langue $language et une réponse $name, When l’acheteur clique « Réserver », Then le refus est expliqué et il reste sur la fiche',
    async ({ status, type, message, language }) => {
      givenArticles(publishedArticle(1));
      reservationAnswers(status, type);
      await openArticleAs(buyer, articleId(1), language);

      await press(
        await screen.findByRole('button', { name: wording[language].reserve }, failureTimeout),
      );
      await waitForReservationRequest();

      expect(
        await screen.findByText(message(wording[language]), {}, failureTimeout),
      ).toBeInTheDocument();
      expect(currentPath()).toBe(articlePath(articleId(1)));
      expect(
        screen.queryByRole('heading', { level: 1, name: wording[language].title }),
      ).not.toBeInTheDocument();
    },
  );
});
