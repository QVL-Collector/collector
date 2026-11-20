import type { Collector } from '../support/collector';
import { expect } from '../support/collector';
import { acheteur, autoriteE2e, clientIdE2e } from '../support/fournisseurIdentite';
import type { Langue } from '../support/langues';
import {
  commandeEmise,
  identifiantArticle,
  ReservationSimulee,
  type CommandeEmise,
} from '../support/reservation';

export const cheminEspaceAcheteur = '/acheteur';

export type TextesEspaceAcheteur = {
  readonly navigation: string;
  readonly titre: string;
  readonly liste: string;
  readonly enAttente: string;
  readonly expiree: string;
  readonly vide: string;
  readonly parcourir: string;
  readonly indisponible: string;
  readonly reessayer: string;
  readonly voir: (titre: string) => string;
};

export const textesEspaceAcheteur: Readonly<Record<Langue, TextesEspaceAcheteur>> = {
  fr: {
    navigation: 'Mes réservations',
    titre: 'Mes réservations',
    liste: 'Réservations',
    enAttente: 'En attente de paiement',
    expiree: 'Expirée',
    vide: 'Aucune réservation pour le moment',
    parcourir: 'Parcourir le catalogue',
    indisponible: 'Vos réservations sont momentanément indisponibles',
    reessayer: 'Réessayer',
    voir: (titre) => `Voir la réservation « ${titre} »`,
  },
  en: {
    navigation: 'My reservations',
    titre: 'My reservations',
    liste: 'Reservations',
    enAttente: 'Awaiting payment',
    expiree: 'Expired',
    vide: 'No reservations yet',
    parcourir: 'Browse the catalogue',
    indisponible: 'Your reservations are temporarily unavailable',
    reessayer: 'Try again',
    voir: (titre) => `View the reservation “${titre}”`,
  },
};

export const montreDeGousset = 'Montre de gousset 1920';
export const carteRare = 'Carte « rare » n°7';

export function dansHeures(heures: number): string {
  return new Date(Date.now() + heures * 3_600_000).toISOString();
}

export function reservationEnAttente(
  rang: number,
  titre: string,
  echeance = dansHeures(2),
): CommandeEmise {
  return { ...commandeEmise(rang, identifiantArticle(rang)), titre, echeance };
}

export function reservationExpiree(rang: number, titre: string): CommandeEmise {
  return { ...reservationEnAttente(rang, titre, dansHeures(-1)), statut: 'expiree' };
}

export function simulationAvec(...commandes: CommandeEmise[]): ReservationSimulee {
  const simulation = new ReservationSimulee();
  simulation.commandes = commandes;
  return simulation;
}

export async function ouvrirEspaceAcheteur(
  collector: Collector,
  simulation: ReservationSimulee | null,
): Promise<void> {
  if (simulation !== null) {
    await simulation.brancher(collector);
  }
  await collector.navigateurEn(collector.langue);
  collector.seConnecterEn(acheteur);
  await collector.ouvrirEspaceProtege(cheminEspaceAcheteur);
  await expect(collector.titrePrincipal()).toHaveText(textesEspaceAcheteur[collector.langue].titre);
}

export async function configurerDemo(collector: Collector, scenario: string): Promise<void> {
  await collector.page.route('**/config.json', (route) =>
    route.fulfill({
      json: {
        marketplaceApiUrl: '/marche',
        catalogue: { source: 'demo', demoScenario: scenario },
        oidc: { authority: autoriteE2e, clientId: clientIdE2e, scope: 'openid profile' },
      },
    }),
  );
}
