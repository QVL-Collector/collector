import { expect } from '@playwright/test';
import { articlesAVerifierDemo, articlesDemo, type Collector } from './collector';
import { acheteur, administrateur } from './fournisseurIdentite';
import { delaiEchecMs, InteractionsSimulees, notificationEmise } from './interactions';

export type Ecran = {
  readonly nom: string;
  readonly ouvrir: (collector: Collector) => Promise<void>;
};

async function catalogueCharge(collector: Collector): Promise<void> {
  await expect(collector.page.getByRole('article').first()).toBeVisible();
}

async function ouvrirArticlesAVerifier(collector: Collector): Promise<void> {
  collector.seConnecterEn(administrateur);
  await collector.ouvrirEspaceProtege('/admin');
  await expect(collector.carteAVerifier(articlesAVerifierDemo.aRefuser)).toBeVisible();
}

const cheminNotifications = '/vendeur/notifications';

async function ouvrirNotificationsRepondant(collector: Collector, statut: number): Promise<void> {
  const interactions = new InteractionsSimulees();
  interactions.statut = statut;
  await interactions.brancher(collector);
  await collector.ouvrirEspaceProtege(cheminNotifications);
}

async function titreAffiche(collector: Collector, titre: string | RegExp): Promise<void> {
  await expect(collector.titrePrincipal()).toHaveText(titre);
}

export const ecrans: readonly Ecran[] = [
  {
    nom: 'catalogue (visiteur)',
    ouvrir: async (collector) => {
      await collector.ouvrir('/');
      await catalogueCharge(collector);
    },
  },
  {
    nom: 'catalogue filtré par catégorie',
    ouvrir: async (collector) => {
      await collector.ouvrir('/?categorie=montres');
      await expect(collector.annonce(collector.textes.annonceCategorieMontres)).toBeAttached();
    },
  },
  {
    nom: 'catalogue, catégorie inconnue',
    ouvrir: async (collector) => {
      await collector.ouvrir('/?categorie=inconnue');
      await expect(
        collector.page.getByRole('heading', { level: 2, name: collector.textes.categorieInconnue }),
      ).toBeVisible();
    },
  },
  {
    nom: 'fiche article',
    ouvrir: async (collector) => {
      await collector.ouvrir(`/articles/${articlesDemo.catalogue}`);
      await expect(collector.page.getByText(collector.textes.vendeurParticulier)).toBeVisible();
    },
  },
  {
    nom: 'fiche article introuvable',
    ouvrir: async (collector) => {
      await collector.ouvrir('/articles/inconnu');
      await titreAffiche(collector, collector.textes.articleIntrouvable);
    },
  },
  {
    nom: 'en-tête connecté (vendeur) sur le catalogue',
    ouvrir: async (collector) => {
      await collector.ouvrir('/');
      await catalogueCharge(collector);
      await collector.seConnecter();
      await catalogueCharge(collector);
    },
  },
  {
    nom: 'connexion échouée',
    ouvrir: async (collector) => {
      await collector.ouvrir('/rappel#error=access_denied&state=inconnu');
      await titreAffiche(collector, collector.textes.echecConnexion);
    },
  },
  {
    nom: 'espace vendeur, liste',
    ouvrir: async (collector) => {
      await collector.ouvrirEspaceProtege('/vendeur');
      await expect(collector.page.getByText(collector.textes.motifIncomplet)).toBeVisible();
    },
  },
  {
    nom: 'espace vendeur, nouveau brouillon',
    ouvrir: async (collector) => {
      await collector.ouvrirEspaceProtege('/vendeur/articles/nouveau');
      await titreAffiche(collector, collector.textes.nouvelArticle);
    },
  },
  {
    nom: 'espace vendeur, formulaire en erreur',
    ouvrir: async (collector) => {
      const { page, textes } = collector;
      await collector.ouvrirEspaceProtege('/vendeur/articles/nouveau');
      await page.getByRole('textbox', { name: textes.champPrix }).fill('abc');
      await page.getByRole('button', { name: textes.creerBrouillon }).click();
      await expect(page.getByText(textes.erreurFormulaire)).toBeVisible();
    },
  },
  {
    nom: 'espace vendeur, article publié (avertissement)',
    ouvrir: async (collector) => {
      await collector.ouvrirEspaceProtege(`/vendeur/articles/${articlesDemo.publie}`);
      await expect(collector.page.getByRole('note')).toContainText(
        collector.textes.avertissementRetrait,
      );
    },
  },
  {
    nom: 'espace vendeur, article refusé (motif)',
    ouvrir: async (collector) => {
      await collector.ouvrirEspaceProtege(`/vendeur/articles/${articlesDemo.refuse}`);
      await expect(collector.etatDeLAnnonce()).toContainText(collector.textes.motifIncomplet);
    },
  },
  {
    nom: 'admin, articles à vérifier',
    ouvrir: ouvrirArticlesAVerifier,
  },
  {
    nom: 'admin, dialogue de refus ouvert',
    ouvrir: async (collector) => {
      await ouvrirArticlesAVerifier(collector);
      const carte = collector.carteAVerifier(articlesAVerifierDemo.aRefuser);
      await collector.boutonDeDecision(carte, collector.textes.refuser).click();
      await expect(collector.page.getByRole('dialog')).toBeVisible();
      await expect(
        collector.page.getByRole('radio', { name: collector.textes.motifPrixAbusif }),
      ).toBeVisible();
    },
  },
  {
    nom: 'admin, décision en conflit (409)',
    ouvrir: async (collector) => {
      await ouvrirArticlesAVerifier(collector);
      const carte = collector.carteAVerifier(articlesAVerifierDemo.modifieParLeVendeur);
      await collector.boutonDeDecision(carte, collector.textes.publier).click();
      await expect(carte.getByText(collector.textes.conflitAdmin)).toBeVisible();
    },
  },
  {
    nom: 'accès réservé',
    ouvrir: async (collector) => {
      collector.seConnecterEn(acheteur);
      await collector.ouvrirEspaceProtege('/vendeur');
      await titreAffiche(collector, collector.textes.accesReserve);
    },
  },
  {
    nom: 'espace vendeur, notifications',
    ouvrir: async (collector) => {
      await collector.ouvrirEspaceProtege(cheminNotifications);
      const region = collector.page.getByRole('region', {
        name: collector.textes.decisionsSurVosArticles,
      });
      await expect(region.getByRole('listitem')).toHaveCount(3);
    },
  },
  {
    nom: 'espace vendeur, notifications des trois issues (API)',
    ouvrir: async (collector) => {
      const interactions = new InteractionsSimulees();
      interactions.notifications = [
        notificationEmise(3, 'refuse', 'article_interdit'),
        notificationEmise(2, 'a_verifier', 'categorie_inconnue'),
        notificationEmise(1),
      ];
      await interactions.brancher(collector);
      await collector.ouvrirEspaceProtege(cheminNotifications);
      await expect(collector.page.getByRole('heading', { level: 3 })).toHaveCount(3);
    },
  },
  {
    nom: 'espace vendeur, aucune notification',
    ouvrir: async (collector) => {
      await ouvrirNotificationsRepondant(collector, 200);
      await expect(
        collector.page.getByRole('heading', { name: collector.textes.aucuneNotification }),
      ).toBeVisible();
    },
  },
  {
    nom: 'espace vendeur, notifications indisponibles',
    ouvrir: async (collector) => {
      await ouvrirNotificationsRepondant(collector, 503);
      await expect(
        collector.page.getByRole('heading', { name: collector.textes.notificationsIndisponibles }),
      ).toBeVisible({ timeout: delaiEchecMs });
    },
  },
  {
    nom: 'espace vendeur, notifications, session expirée (401)',
    ouvrir: async (collector) => {
      await ouvrirNotificationsRepondant(collector, 401);
      await expect(
        collector.page.getByRole('heading', { name: collector.textes.sessionExpiree }),
      ).toBeVisible();
    },
  },
  {
    nom: 'espace vendeur, notifications, accès réservé (403)',
    ouvrir: async (collector) => {
      await ouvrirNotificationsRepondant(collector, 403);
      await expect(
        collector.page.getByRole('heading', { name: collector.textes.accesReserve }),
      ).toBeVisible();
    },
  },
  {
    nom: 'page introuvable',
    ouvrir: async (collector) => {
      await collector.ouvrir('/page-inexistante');
      await titreAffiche(collector, collector.textes.pageIntrouvable);
    },
  },
  {
    nom: 'échec de démarrage',
    ouvrir: async (collector) => {
      await collector.configurationIndisponible();
      await collector.ouvrir('/');
      await expect(collector.page.getByText(collector.textes.echecDemarrage)).toBeVisible();
    },
  },
];
