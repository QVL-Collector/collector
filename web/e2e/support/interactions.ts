import type { Route } from '@playwright/test';
import type { Collector } from './collector';
import { autoriteE2e, clientIdE2e } from './fournisseurIdentite';

export type IssueDecision = 'publie' | 'refuse' | 'a_verifier';

export type NotificationEmise = {
  readonly id: string;
  readonly articleId: string;
  readonly revision: number;
  readonly issue: IssueDecision;
  readonly motif: string | null;
  readonly decideLe: string;
};

export type LectureNotifications = {
  readonly url: URL;
  readonly autorisation: string | null;
};

const configurationApi = {
  marketplaceApiUrl: '/marche',
  interactionsApiUrl: '/interactions',
  catalogue: { source: 'api', demoScenario: 'nominal' },
  oidc: { authority: autoriteE2e, clientId: clientIdE2e, scope: 'openid profile' },
};

export const delaiMaximalMs = 10_000;
export const delaiEchecMs = 15_000;

export function notificationEmise(
  rang: number,
  issue: IssueDecision = 'publie',
  motif: string | null = null,
): NotificationEmise {
  const suffixe = rang.toString().padStart(12, '0');
  return {
    id: `00000000-0000-4000-9000-${suffixe}`,
    articleId: `00000000-0000-4000-a000-${suffixe}`,
    revision: 2,
    issue,
    motif,
    decideLe: new Date(Date.UTC(2026, 10, 4, 9, rang)).toISOString(),
  };
}

function probleme(statut: number) {
  return { type: 'about:blank', title: 'Problème', status: statut };
}

export class InteractionsSimulees {
  notifications: NotificationEmise[] = [];
  statut = 200;
  readonly lectures: LectureNotifications[] = [];

  async brancher(collector: Collector): Promise<void> {
    const { page } = collector;
    await page.route('**/config.json', (route) => route.fulfill({ json: configurationApi }));
    await page.route('**/marche/**', (route) =>
      route.fulfill({ status: 404, json: probleme(404), contentType: 'application/problem+json' }),
    );
    await page.route('**/interactions/v1/notifications**', (route) => this.repondre(route));
  }

  recevoir(notification: NotificationEmise): void {
    this.notifications = [notification, ...this.notifications];
  }

  private repondre(route: Route): Promise<void> {
    const requete = route.request();
    this.lectures.push({
      url: new URL(requete.url()),
      autorisation: requete.headers().authorization ?? null,
    });
    if (this.statut !== 200) {
      return route.fulfill({
        status: this.statut,
        contentType: 'application/problem+json',
        body: JSON.stringify(probleme(this.statut)),
      });
    }
    return route.fulfill({
      json: {
        elements: this.notifications,
        page: 1,
        taille: 20,
        total: this.notifications.length,
      },
    });
  }
}
