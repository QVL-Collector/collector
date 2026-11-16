import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { buyer, press } from '../US-008/identityProviderHarness';
import {
  articleId,
  failureTimeout,
  givenArticles,
  givenNextOrder,
  openArticleAs,
  order,
  publishedArticle,
  waitForReservationRequest,
  wording,
} from './reservationHarness';

async function reserveArticleOne(): Promise<void> {
  givenArticles(publishedArticle(1, { prix: { montantCentimes: 1, devise: 'EUR' } }));
  givenNextOrder(
    order(1, articleId(1), {
      prix: 12000,
      frais: 590,
      total: 12590,
      commission: 630,
      vendeur: 11960,
    }),
  );
  await openArticleAs(buyer, articleId(1));
  await press(await screen.findByRole('button', { name: wording.fr.reserve }, failureTimeout));
}

describe('US-030-AC6 : la demande de l’acheteur ne porte aucun prix ni montant', () => {
  it('Given un acheteur sur la fiche, When il réserve, Then le corps envoyé ne contient que l’identifiant de l’article', async () => {
    await reserveArticleOne();

    const sent = await waitForReservationRequest();

    expect(JSON.parse(sent.body)).toStrictEqual({ articleId: articleId(1) });
  });

  it('Given un acheteur sur la fiche, When il réserve, Then la demande est un JSON sans prix, montant, frais ni identité d’acheteur', async () => {
    await reserveArticleOne();

    const sent = await waitForReservationRequest();

    expect(sent.contentType).toMatch(/^application\/json/);
    expect(sent.body).not.toMatch(/prix|montant|frais|commission|total|acheteur|vendeur|sub/i);
  });

  it('Given un prix affiché sur la fiche différent de celui du serveur, When la commande est créée, Then l’écran montre les montants renvoyés par Transactions', async () => {
    await reserveArticleOne();

    await screen.findByRole('heading', { level: 1, name: wording.fr.title }, failureTimeout);

    expect(
      screen.getByText('Prix de référence', { selector: 'dt' }).nextElementSibling?.textContent,
    ).toMatch(/120,00/);
  });
});
