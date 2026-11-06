import { expect, test } from '../support/collector';
import { textes } from '../support/textes';

const { fr, en } = textes;

test.describe('US-025-AC1 passage en anglais par le sélecteur, sans rechargement', () => {
  test('espace vendeur : textes clés, message d’erreur et motif de refus en anglais', async ({
    collector,
  }) => {
    const { page } = collector;
    await collector.ouvrirEspaceProtege('/vendeur/articles/nouveau');
    await expect(collector.titrePrincipal()).toHaveText(fr.nouvelArticle);
    await page.getByRole('textbox', { name: fr.champPrix }).fill('abc');
    await page.getByRole('button', { name: fr.creerBrouillon }).click();
    await expect(page.getByText(fr.erreurMontant)).toBeVisible();
    await expect(page.getByText(fr.erreurFormulaire)).toBeVisible();
    const chargements = collector.chargementsDeConfiguration;
    await collector.poserMarqueurSansRechargement();

    await collector.choisirLangue('en');

    await expect(collector.titrePrincipal()).toHaveText(en.nouvelArticle);
    await expect(page).toHaveTitle('New item · Collector');
    await expect(page.getByRole('textbox', { name: en.champPrix })).toHaveValue('abc');
    await expect(page.getByText(en.erreurMontant)).toBeVisible();
    await expect(page.getByText(en.erreurFormulaire)).toBeVisible();
    await expect(page.getByRole('button', { name: en.creerBrouillon })).toBeVisible();
    await expect(page.getByRole('banner').getByText(en.roleVendeur, { exact: true })).toBeVisible();
    await expect(page.getByRole('button', { name: en.seDeconnecter })).toBeVisible();

    await page.getByRole('link', { name: en.retourMesArticles }).click();
    await expect(collector.titrePrincipal()).toHaveText(en.espaceVendeur);
    await expect(page.getByRole('heading', { level: 2, name: en.mesArticles })).toBeVisible();
    const refuse = page.getByRole('link', { name: /Lot de douze figurines/ });
    await expect(refuse).toContainText(en.refuse);
    await expect(refuse).toContainText(en.motifIncomplet);
    await expect(page.getByText(fr.motifIncomplet)).toHaveCount(0);

    expect(await collector.marqueurSansRechargementPresent()).toBe(true);
    expect(collector.chargementsDeConfiguration).toBe(chargements);
  });

  test('fiche introuvable en anglais, puis retour au catalogue', async ({ collector }) => {
    const { page } = collector;
    await collector.ouvrir('/articles/inconnu');
    await expect(collector.titrePrincipal()).toHaveText(fr.articleIntrouvable);
    await collector.poserMarqueurSansRechargement();

    await collector.choisirLangue('en');

    await expect(collector.titrePrincipal()).toHaveText(en.articleIntrouvable);
    await expect(page.getByRole('button', { name: en.seConnecter })).toBeVisible();
    await page.getByRole('link', { name: en.retourCatalogue }).click();
    await expect(collector.titrePrincipal()).toHaveText(en.catalogue);
    expect(await collector.marqueurSansRechargementPresent()).toBe(true);
  });
});
