import { tabulerJusqua } from '../support/accessibilite';
import { expect, test } from '../support/collector';

test.describe('US-025-AC5 parcours de mise en vente au clavier seul', () => {
  test('connexion, espace vendeur, brouillon, soumission : focus visible à chaque tabulation', async ({
    collector,
    langue,
  }) => {
    const { page, textes } = collector;
    const clavier = page.keyboard;
    await collector.navigateurEn(langue);
    await collector.ouvrir('/');
    await expect(page.getByRole('article').first()).toBeVisible();

    await tabulerJusqua(page, page.getByRole('button', { name: textes.seConnecter }));
    await clavier.press('Enter');
    await collector.attendreSessionOuverte();
    await expect(page.getByRole('article').first()).toBeVisible();
    await expect(collector.titrePrincipal()).toBeFocused();

    await tabulerJusqua(
      page,
      page.getByRole('banner').getByRole('link', { name: textes.espaceVendeur }),
      'arriere',
    );
    await clavier.press('Enter');
    await expect(collector.titrePrincipal()).toHaveText(textes.espaceVendeur);
    await expect(collector.titrePrincipal()).toBeFocused();

    await tabulerJusqua(page, page.getByRole('link', { name: textes.nouvelArticle }));
    await clavier.press('Enter');
    await expect(collector.titrePrincipal()).toHaveText(textes.nouvelArticle);
    await expect(collector.titrePrincipal()).toBeFocused();

    const titre = `Figurine en plomb, clavier ${langue}`;
    await tabulerJusqua(page, page.getByRole('textbox', { name: textes.champTitre }));
    await clavier.type(titre);
    await tabulerJusqua(page, page.getByRole('textbox', { name: textes.champDescription }));
    await clavier.type('Peinte à la main, socle d’origine.');
    const categorie = page.getByRole('combobox', { name: textes.champCategorie });
    await tabulerJusqua(page, categorie);
    await clavier.type('Figurines');
    await expect(categorie).toHaveValue('figurines');
    await tabulerJusqua(page, page.getByRole('textbox', { name: textes.champPrix }));
    await clavier.type(textes.prixSaisi);
    await tabulerJusqua(page, page.getByRole('button', { name: textes.creerBrouillon }));
    await clavier.press('Enter');

    await expect(collector.titrePrincipal()).toHaveText(titre);
    const etat = collector.etatDeLAnnonce();
    await expect(etat.getByText(textes.brouillon, { exact: true })).toBeVisible();
    await tabulerJusqua(page, etat.getByRole('button', { name: textes.soumettre }));
    await clavier.press('Space');
    await expect(etat.getByText(textes.publie, { exact: true })).toBeVisible({
      timeout: 15_000,
    });
  });
});
