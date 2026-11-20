import type { Route } from '@playwright/test';
import type { Collector } from './collector';
import { autoriteE2e, clientIdE2e } from './fournisseurIdentite';
import type { Langue } from './langues';

type Montant = { readonly montantCentimes: number; readonly devise: 'EUR' };

export type ArticleEnVente = {
  readonly id: string;
  readonly titre: string;
  readonly prix: Montant;
  readonly fraisDePort: Montant | null;
  reserve: boolean;
};

export type CommandeEmise = {
  readonly id: string;
  readonly articleId: string;
  readonly titre: string;
  readonly statut: 'en_attente_paiement' | 'expiree';
  readonly prix: Montant;
  readonly fraisDePort: Montant;
  readonly totalPaye: Montant;
  readonly commission: Montant;
  readonly montantVendeur: Montant;
  readonly echeance: string;
  readonly creeLe: string;
};

export type DemandeDeCommande = {
  readonly autorisation: string | null;
  readonly typeDeContenu: string | null;
  readonly corps: string;
};

const configurationApi = {
  marketplaceApiUrl: '/marche',
  interactionsApiUrl: '/interactions',
  transactionsApiUrl: '/transactions',
  catalogue: { source: 'api', demoScenario: 'nominal' },
  oidc: { authority: autoriteE2e, clientId: clientIdE2e, scope: 'openid profile' },
};

const cheminCommandes = '/transactions/v1/commandes';
const categorie = { id: 'figurines', libelle: 'Figurines' };

export const echeanceE2e = '2026-11-16T10:25:00.123456Z';

function euros(centimes: number): Montant {
  return { montantCentimes: centimes, devise: 'EUR' };
}

export function identifiantArticle(rang: number): string {
  return `00000000-0000-4000-8000-${rang.toString().padStart(12, '0')}`;
}

export function identifiantCommande(rang: number): string {
  return `00000000-0000-4000-b000-${rang.toString().padStart(12, '0')}`;
}

export function titreDeLArticle(id: string): string {
  return `Figurine de collection ${String(Number(id.slice(-12)))}`;
}

export function articleEnVente(rang: number, reserve = false): ArticleEnVente {
  return {
    id: identifiantArticle(rang),
    titre: titreDeLArticle(identifiantArticle(rang)),
    prix: euros(12000),
    fraisDePort: euros(590),
    reserve,
  };
}

export type Montants = {
  readonly prix: number;
  readonly frais: number;
  readonly total: number;
  readonly commission: number;
  readonly vendeur: number;
};

export const montantsNominaux: Montants = {
  prix: 12000,
  frais: 590,
  total: 12590,
  commission: 630,
  vendeur: 11960,
};

export function commandeEmise(
  rang: number,
  articleId: string,
  montants: Montants = montantsNominaux,
): CommandeEmise {
  return {
    id: identifiantCommande(rang),
    articleId,
    titre: titreDeLArticle(articleId),
    statut: 'en_attente_paiement',
    prix: euros(montants.prix),
    fraisDePort: euros(montants.frais),
    totalPaye: euros(montants.total),
    commission: euros(montants.commission),
    montantVendeur: euros(montants.vendeur),
    echeance: echeanceE2e,
    creeLe: '2026-11-16T10:15:00.234567Z',
  };
}

export function enEuros(centimes: number, langue: Langue): string {
  return new Intl.NumberFormat(langue === 'fr' ? 'fr-FR' : 'en-GB', {
    style: 'currency',
    currency: 'EUR',
  })
    .format(centimes / 100)
    .replace(/\s+/g, ' ');
}

function entree(article: ArticleEnVente) {
  return {
    id: article.id,
    titre: article.titre,
    prix: article.prix,
    categorie,
    reserve: article.reserve,
  };
}

function probleme(statut: number, type = 'about:blank') {
  return { type, title: 'Problème', status: statut };
}

export class ReservationSimulee {
  articles: ArticleEnVente[] = [];
  commandes: CommandeEmise[] = [];
  prochaineCommande: CommandeEmise | null = null;
  refus: { statut: number; type: string } | null = null;
  listeIndisponible = false;
  readonly demandes: DemandeDeCommande[] = [];
  readonly listes: URL[] = [];

  async brancher(collector: Collector): Promise<void> {
    const { page } = collector;
    await page.route('**/config.json', (route) => route.fulfill({ json: configurationApi }));
    await page.route('**/marche/**', (route) => this.repondreMarche(route));
    await page.route('**/transactions/**', (route) => this.repondreTransactions(route));
  }

  private repondreMarche(route: Route): Promise<void> {
    const url = new URL(route.request().url());
    if (url.pathname === '/marche/v1/categories') {
      return route.fulfill({ json: { elements: [categorie] } });
    }
    if (url.pathname === '/marche/v1/articles') {
      return route.fulfill({
        json: {
          elements: this.articles.map(entree),
          page: 1,
          taille: 12,
          total: this.articles.length,
        },
      });
    }
    const id = decodeURIComponent(url.pathname.split('/').at(-1) ?? '');
    const article = this.articles.find((candidat) => candidat.id === id);
    if (article === undefined) {
      return route.fulfill({
        status: 404,
        contentType: 'application/problem+json',
        body: JSON.stringify(probleme(404)),
      });
    }
    return route.fulfill({
      json: {
        ...entree(article),
        description: 'Figurine en résine peinte à la main.',
        fraisDePort: article.fraisDePort,
      },
    });
  }

  private repondreTransactions(route: Route): Promise<void> {
    const requete = route.request();
    const url = new URL(requete.url());
    if (requete.method() === 'POST' && url.pathname === cheminCommandes) {
      return this.creer(route);
    }
    if (requete.method() === 'GET' && url.pathname === cheminCommandes) {
      return this.lister(route, url);
    }
    const id = decodeURIComponent(url.pathname.split('/').at(-1) ?? '');
    const commande = this.commandes.find((candidate) => candidate.id === id);
    if (requete.method() === 'GET' && commande !== undefined) {
      return route.fulfill({ json: commande });
    }
    return route.fulfill({
      status: 404,
      contentType: 'application/problem+json',
      body: JSON.stringify(probleme(404)),
    });
  }

  private lister(route: Route, url: URL): Promise<void> {
    this.listes.push(url);
    if (this.listeIndisponible) {
      return route.fulfill({
        status: 503,
        contentType: 'application/problem+json',
        body: JSON.stringify(probleme(503)),
      });
    }
    const page = Number(url.searchParams.get('page') ?? '1');
    const taille = Number(url.searchParams.get('taille') ?? '20');
    return route.fulfill({
      json: {
        elements: this.commandes.slice((page - 1) * taille, page * taille),
        page,
        taille,
        total: this.commandes.length,
      },
    });
  }

  private creer(route: Route): Promise<void> {
    const requete = route.request();
    const entetes = requete.headers();
    const corps = requete.postData() ?? '';
    this.demandes.push({
      autorisation: entetes.authorization ?? null,
      typeDeContenu: entetes['content-type'] ?? null,
      corps,
    });
    if (this.refus !== null) {
      return route.fulfill({
        status: this.refus.statut,
        contentType: 'application/problem+json',
        body: JSON.stringify(probleme(this.refus.statut, this.refus.type)),
      });
    }
    const commande = this.prochaineCommande;
    if (commande === null) {
      return route.fulfill({
        status: 503,
        contentType: 'application/problem+json',
        body: JSON.stringify(probleme(503)),
      });
    }
    this.commandes.push(commande);
    const reservee = this.articles.find((article) => article.id === commande.articleId);
    if (reservee !== undefined) {
      reservee.reserve = true;
    }
    return route.fulfill({
      status: 201,
      headers: { Location: `${cheminCommandes}/${commande.id}` },
      json: commande,
    });
  }
}
