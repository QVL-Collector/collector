import { articlesDemo, contenant, expect, test } from '../support/collector';

test.describe('US-025-AC3 parcours de mise en vente', () => {
  test('catalogue, filtre, fiche, connexion vendeur, brouillon, soumission, état publié', async ({
    collector,
    langue,
  }) => {
    const { page, textes } = collector;
    await collector.ouvrir('/');
    await collector.choisirLangue(langue);
    await expect(collector.titrePrincipal()).toHaveText(textes.catalogue);

    await page.getByRole('combobox', { name: textes.filtreCategorie }).selectOption('montres');
    await expect(page).toHaveURL(/[?&]categorie=montres/);
    await expect(collector.annonce(textes.annonceCategorieMontres)).toBeAttached();

    const carte = page.getByRole('article').first();
    const titreArticle = (await carte.getByRole('heading').textContent()) ?? '';
    expect(titreArticle).not.toBe('');
    await carte.getByRole('link').click();
    await expect(collector.titrePrincipal()).toHaveText(titreArticle);
    await expect(page.getByText(textes.vendeurParticulier)).toBeVisible();

    await collector.seConnecter();
    await expect(collector.titrePrincipal()).toHaveText(titreArticle);
    await expect(page.locator('html')).toHaveAttribute('lang', langue);
    await expect(
      page.getByRole('banner').getByText(textes.roleVendeur, { exact: true }),
    ).toBeVisible();

    await page.getByRole('banner').getByRole('link', { name: textes.espaceVendeur }).click();
    await expect(collector.titrePrincipal()).toHaveText(textes.espaceVendeur);
    await page.getByRole('link', { name: textes.nouvelArticle }).click();
    await expect(collector.titrePrincipal()).toHaveText(textes.nouvelArticle);

    const titre = `Montre de gousset, parcours ${langue}`;
    await page.getByRole('textbox', { name: textes.champTitre }).fill(titre);
    await page
      .getByRole('textbox', { name: textes.champDescription })
      .fill('Boîtier argent, remontoir d’origine.');
    await page.getByRole('combobox', { name: textes.champCategorie }).selectOption('figurines');
    await page.getByRole('textbox', { name: textes.champPrix }).fill(textes.prixSaisi);
    await page.getByRole('textbox', { name: textes.champFraisDePort }).fill('6');
    await page.getByRole('button', { name: textes.creerBrouillon }).click();

    await expect(collector.titrePrincipal()).toHaveText(titre);
    const etat = collector.etatDeLAnnonce();
    await expect(etat.getByText(textes.brouillon, { exact: true })).toBeVisible();
    await etat.getByRole('button', { name: textes.soumettre }).click();
    await expect(etat.getByText(textes.enControle, { exact: true })).toBeVisible();
    await expect(etat.getByText(textes.publie, { exact: true })).toBeVisible({
      timeout: 15_000,
    });

    await page.getByRole('link', { name: textes.retourMesArticles }).click();
    await expect(page.getByRole('link', { name: contenant(titre) })).toContainText(textes.publie);
  });

  test('modification d’un article publié : avertissement puis retour au contrôle (US-021)', async ({
    collector,
    langue,
  }) => {
    const { page, textes } = collector;
    await collector.ouvrir('/');
    await collector.choisirLangue(langue);
    await collector.seConnecter();
    await page.getByRole('banner').getByRole('link', { name: textes.espaceVendeur }).click();
    await page.getByRole('link', { name: /Affiche de cinéma originale, 1962/ }).click();
    await expect(page).toHaveURL(new RegExp(`/vendeur/articles/${articlesDemo.publie}$`));

    const etat = collector.etatDeLAnnonce();
    await expect(etat.getByText(textes.publie, { exact: true })).toBeVisible();
    await expect(page.getByRole('note')).toContainText(textes.avertissementRetrait);
    const enregistrer = page.getByRole('button', { name: textes.enregistrerEtRenvoyer });
    await expect(enregistrer).toHaveAccessibleDescription(contenant(textes.avertissementRetrait));

    await page
      .getByRole('textbox', { name: textes.champTitre })
      .fill('Affiche de cinéma originale, 1962, entoilée');
    await enregistrer.click();

    await expect(page.getByText(textes.renvoyeAuControle).first()).toBeVisible();
    await expect(etat.getByText(textes.enControle, { exact: true })).toBeVisible();
    await expect(etat.getByText(textes.publie, { exact: true })).toBeVisible({
      timeout: 15_000,
    });
  });
});
