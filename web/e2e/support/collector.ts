import { test as base, expect, type Locator, type Page } from '@playwright/test';
import {
  autoriteE2e,
  clientIdE2e,
  FournisseurIdentite,
  vendeur,
  type CompteFictif,
} from './fournisseurIdentite';
import { localeNavigateur, type Langue } from './langues';
import { textes, type Textes } from './textes';

export const configurationDemo = {
  marketplaceApiUrl: '/marche',
  catalogue: { source: 'demo', demoScenario: 'nominal' },
  oidc: { authority: autoriteE2e, clientId: clientIdE2e, scope: 'openid profile' },
};

export const articlesDemo = {
  catalogue: 'demo-001',
  publie: 'c2e9b4a7-1d6f-4f80-9a3b-8c5e7d2f1a03',
  refuse: '7f4a1d2e-9c8b-4e57-b3a0-6e2f8d1c4b02',
} as const;

const marqueurSansRechargement = 'collectorE2eSansRechargement';

export function contenant(texte: string): RegExp {
  return new RegExp(texte.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));
}

export class Collector {
  readonly fournisseur = new FournisseurIdentite();
  private chargementsConfiguration = 0;

  constructor(
    readonly page: Page,
    readonly langue: Langue,
  ) {}

  get textes(): Textes {
    return textes[this.langue];
  }

  get chargementsDeConfiguration(): number {
    return this.chargementsConfiguration;
  }

  async demarrer(): Promise<void> {
    await this.fournisseur.brancher(this.page);
    await this.page.route('**/config.json', (route) => {
      this.chargementsConfiguration += 1;
      return route.fulfill({ json: configurationDemo });
    });
  }

  async configurationIndisponible(): Promise<void> {
    await this.page.route('**/config.json', (route) => route.fulfill({ status: 503, body: '' }));
  }

  async navigateurEn(langue: Langue): Promise<void> {
    const locale = localeNavigateur[langue];
    await this.page.addInitScript((tag) => {
      Object.defineProperty(Navigator.prototype, 'languages', { get: () => [tag] });
      Object.defineProperty(Navigator.prototype, 'language', { get: () => tag });
    }, locale);
  }

  seConnecterEn(compte: CompteFictif): void {
    this.fournisseur.compte = compte;
  }

  async ouvrir(chemin: string): Promise<void> {
    await this.page.goto(chemin);
  }

  titrePrincipal(): Locator {
    return this.page.getByRole('heading', { level: 1 });
  }

  async choisirLangue(langue: Langue): Promise<void> {
    const choix = this.page.getByRole('button', { name: textes[langue].nomLangue, exact: true });
    await choix.click();
    await expect(choix).toHaveAttribute('aria-pressed', 'true');
    await expect(this.page.locator('html')).toHaveAttribute('lang', langue);
  }

  async seConnecter(): Promise<void> {
    await this.page.getByRole('button', { name: this.textes.seConnecter }).click();
    await this.attendreSessionOuverte();
  }

  async attendreSessionOuverte(): Promise<void> {
    await expect(this.page.getByRole('button', { name: this.textes.seDeconnecter })).toBeVisible();
  }

  async ouvrirEspaceProtege(chemin: string): Promise<void> {
    await this.ouvrir(chemin);
    await this.page.waitForURL((url) => url.pathname === chemin);
    await this.attendreSessionOuverte();
  }

  async poserMarqueurSansRechargement(): Promise<void> {
    await this.page.evaluate((nom) => {
      Object.defineProperty(window, nom, { value: true });
    }, marqueurSansRechargement);
  }

  async marqueurSansRechargementPresent(): Promise<boolean> {
    return this.page.evaluate((nom) => nom in window, marqueurSansRechargement);
  }

  annonce(texte: RegExp): Locator {
    return this.page.getByRole('status').filter({ hasText: texte }).first();
  }

  etatDeLAnnonce(): Locator {
    return this.page.getByRole('region', { name: this.textes.etatAnnonce });
  }
}

type OptionsCollector = { readonly langue: Langue };

export type { OptionsCollector };

export const test = base.extend<OptionsCollector & { readonly collector: Collector }>({
  langue: ['fr', { option: true }],
  collector: async ({ page, langue }, fournir) => {
    const collector = new Collector(page, langue);
    await collector.demarrer();
    collector.seConnecterEn(vendeur);
    await fournir(collector);
  },
});

export { expect };
