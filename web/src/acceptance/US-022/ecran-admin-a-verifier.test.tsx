import { screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import {
  axeTimeout,
  blockingViolations,
  expectAnnounced,
} from '../US-013/sellerMarketplaceHarness';
import {
  accounts,
  adminScreenPath,
  administration,
  changeRevision,
  decisions,
  givenToReview,
  listReads,
  noLongerToReview,
  openAdminScreenWith,
  scriptNextDecision,
  toReview,
  wording,
  type RefusalReason,
} from './adminReviewHarness';
import { signInAs } from '../US-008/identityProviderHarness';

const first = toReview(1);
const second = toReview(2, {
  titre: 'Timbre rare 1849',
  description: 'Cérès noir sur jaune',
  categorie: { id: 'timbres', libelle: 'Timbres' },
  prix: { montantCentimes: 98000, devise: 'EUR' },
  fraisDePort: null,
  motif: 'vendeur_a_verifier',
  revision: 5,
  vendeurId: 'vendeur-timbres',
});
const markupTitle =
  '<script>window.__us022 = true</script><b>Gras</b> & <img src=x onerror="window.__us022 = true">';

function card(title: string): HTMLElement {
  const heading = screen.getByRole('heading', { level: 3, name: title });
  const item = heading.closest('li');
  if (item === null) {
    throw new Error(`Carte de « ${title} » introuvable`);
  }
  return item;
}

function shownTitles(): string[] {
  return within(screen.getByRole('list', { name: 'Annonces à vérifier' }))
    .getAllByRole('heading', { level: 3 })
    .map((heading) => heading.textContent);
}

async function publish(title: string): Promise<void> {
  await userEvent.click(screen.getByRole('button', { name: wording.publish(title) }));
}

async function openRefusal(title: string): Promise<HTMLElement> {
  await userEvent.click(screen.getByRole('button', { name: wording.refuse(title) }));
  return screen.findByRole('dialog', { name: wording.refusalTitle });
}

async function refuse(title: string, reason: RefusalReason): Promise<void> {
  const dialog = await openRefusal(title);
  await userEvent.click(
    within(dialog).getByRole('radio', { name: wording.refusalReasons[reason] }),
  );
  await userEvent.click(within(dialog).getByRole('button', { name: wording.confirm }));
}

async function expectConflictShown(): Promise<void> {
  expect(await screen.findByText(wording.conflict)).toBeInTheDocument();
  expect(await screen.findByRole('button', { name: wording.reload })).toBeInTheDocument();
}

async function closeRefusal(dialog: HTMLElement): Promise<void> {
  await userEvent.click(within(dialog).getByRole('button', { name: wording.cancel }));
  await vi.waitFor(() => {
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
}

describe('US-022 AC1 liste des articles à vérifier', () => {
  it('Given des articles à vérifier, When l’admin ouvre l’écran, Then chacun montre le contenu de la révision examinée et le motif du contrôle, dans l’ordre reçu', async () => {
    givenToReview(second, first);

    await openAdminScreenWith(2);

    expect(shownTitles()).toEqual([first.titre, second.titre]);
    const timbre = within(card(second.titre));
    expect(timbre.getByText(second.description)).toBeInTheDocument();
    expect(timbre.getByText('Timbres')).toBeInTheDocument();
    expect(timbre.getByText(/980,00\s€/)).toBeInTheDocument();
    expect(
      timbre.getByText(wording.controlReason(wording.controlReasons.vendeur_a_verifier)),
    ).toBeInTheDocument();
    expect(timbre.getByText(wording.revision(5))).toBeInTheDocument();
    expect(timbre.getByText(second.vendeurId)).toBeInTheDocument();
    const figurine = within(card(first.titre));
    expect(figurine.getByText(first.description)).toBeInTheDocument();
    expect(
      figurine.getByText(wording.controlReason(wording.controlReasons.categorie_inconnue)),
    ).toBeInTheDocument();
    expect(figurine.getByText(/4,00\s€/)).toBeInTheDocument();
    await expectAnnounced(wording.loaded(2));
  });

  it.each(Object.entries(wording.controlReasons))(
    'Given un article à vérifier pour le motif %s, When l’admin ouvre l’écran, Then le motif « %s » est affiché',
    async (motif, label) => {
      givenToReview(toReview(1, { motif: motif as keyof typeof wording.controlReasons }));

      await openAdminScreenWith(1);

      expect(within(card(first.titre)).getByText(wording.controlReason(label))).toBeInTheDocument();
    },
  );

  it('Given un article à vérifier sans catégorie ni prix, When l’admin ouvre l’écran, Then la carte reste affichée avec ses actions', async () => {
    givenToReview(toReview(1, { categorie: null, prix: null, fraisDePort: null, description: '' }));

    await openAdminScreenWith(1);

    expect(
      within(card(first.titre)).getByRole('button', { name: wording.publish(first.titre) }),
    ).toBeEnabled();
  });

  it('Given aucun article à vérifier, When l’admin ouvre l’écran, Then un état vide est affiché', async () => {
    await openAdminScreenWith(0);

    expect(screen.queryByRole('button', { name: /^Publier « / })).not.toBeInTheDocument();
  });

  it('Given l’admin connecté, When l’écran charge la liste, Then l’appel porte son jeton', async () => {
    givenToReview(first);

    await openAdminScreenWith(1);

    expect(listReads().length).toBeGreaterThan(0);
    expect(listReads().every((request) => request.authorization?.startsWith('Bearer '))).toBe(true);
  });
});

describe('US-022 AC2 publier ou refuser avec un motif de la liste fermée', () => {
  it('Given un article de la liste, When l’admin le publie, Then la décision porte sur la révision affichée et l’article quitte la liste', async () => {
    givenToReview(first, second);
    await openAdminScreenWith(2);

    await publish(second.titre);

    await expectAnnounced(wording.published(second.titre));
    expect(decisions().map((request) => [request.path, request.body])).toEqual([
      [`/marche/v1/admin/articles/${second.id}/decision`, { revision: 5, decision: 'publier' }],
    ]);
    await vi.waitFor(() => {
      expect(
        screen.queryByRole('heading', { level: 3, name: second.titre }),
      ).not.toBeInTheDocument();
    });
    expect(administration().articles.get(second.id)?.etat).toBe('publie');
    expect(shownTitles()).toEqual([first.titre]);
  });

  it('Given le dialogue de refus, When il s’ouvre, Then il propose exactement les cinq motifs de la liste fermée', async () => {
    givenToReview(first);
    await openAdminScreenWith(1);

    const dialog = await openRefusal(first.titre);

    const group = within(dialog).getByRole('radiogroup', { name: wording.reasonLabel });
    expect(
      within(group)
        .getAllByRole('radio')
        .map((radio) => radio.getAttribute('value')),
    ).toEqual(Object.keys(wording.refusalReasons));
    for (const label of Object.values(wording.refusalReasons)) {
      expect(within(group).getByRole('radio', { name: label })).not.toBeChecked();
    }
  });

  it.each(Object.keys(wording.refusalReasons) as RefusalReason[])(
    'Given un article de la liste, When l’admin le refuse avec le motif %s, Then la décision envoie ce motif et l’article quitte la liste',
    async (reason) => {
      givenToReview(first);
      await openAdminScreenWith(1);

      await refuse(first.titre, reason);

      await expectAnnounced(wording.refused(first.titre));
      expect(decisions().map((request) => request.body)).toEqual([
        { revision: first.revision, decision: 'refuser', motif: reason },
      ]);
      expect(administration().articles.get(first.id)).toMatchObject({
        etat: 'refuse',
        motif: reason,
      });
      await screen.findByRole('heading', { level: 2, name: wording.empty });
    },
  );

  it('Given le dialogue de refus, When l’admin confirme sans motif, Then rien n’est envoyé et un motif est demandé', async () => {
    givenToReview(first);
    await openAdminScreenWith(1);
    const dialog = await openRefusal(first.titre);

    await userEvent.click(within(dialog).getByRole('button', { name: wording.confirm }));

    expect(await within(dialog).findByText(wording.missingReason)).toBeInTheDocument();
    expect(decisions()).toEqual([]);
  });

  it('Given le dialogue de refus, When l’admin annule, Then rien n’est envoyé et l’article reste dans la liste', async () => {
    givenToReview(first);
    await openAdminScreenWith(1);
    const dialog = await openRefusal(first.titre);
    await userEvent.click(
      within(dialog).getByRole('radio', { name: wording.refusalReasons.prix_abusif }),
    );

    await closeRefusal(dialog);

    expect(decisions()).toEqual([]);
    expect(shownTitles()).toEqual([first.titre]);
  });
});

describe('US-022 AC3 décision sur une révision périmée', () => {
  it('Given l’article passé en révision N+1 depuis l’affichage, When l’admin le publie, Then 409 et l’écran invite à recharger la liste', async () => {
    givenToReview(first);
    await openAdminScreenWith(1);
    changeRevision(first.id);

    await publish(first.titre);

    await expectConflictShown();
    expect(decisions().map((request) => request.body)).toEqual([
      { revision: first.revision, decision: 'publier' },
    ]);
    expect(administration().articles.get(first.id)).toMatchObject({
      etat: 'a_verifier',
      revision: first.revision + 1,
    });
  });

  it('Given l’article passé en révision N+1, When l’admin le refuse, Then 409 et l’écran invite à recharger la liste', async () => {
    givenToReview(first);
    await openAdminScreenWith(1);
    changeRevision(first.id);

    await refuse(first.titre, 'article_interdit');

    await expectConflictShown();
    expect(administration().articles.get(first.id)?.etat).toBe('a_verifier');
  });

  it('Given l’article déjà tranché ailleurs, When l’admin décide, Then l’écran invite aussi à recharger la liste', async () => {
    givenToReview(first);
    await openAdminScreenWith(1);
    scriptNextDecision(noLongerToReview());

    await publish(first.titre);

    await expectConflictShown();
  });

  it('Given l’invitation à recharger, When l’admin recharge, Then la liste montre la nouvelle révision et la décision suivante la porte', async () => {
    givenToReview(first);
    await openAdminScreenWith(1);
    changeRevision(first.id);
    await publish(first.titre);
    await expectConflictShown();
    const readsBefore = listReads().length;
    const renamed = `${first.titre} (modifiée)`;

    await userEvent.click(screen.getByRole('button', { name: wording.reload }));

    await screen.findByRole('heading', { level: 3, name: renamed });
    expect(listReads().length).toBeGreaterThan(readsBefore);
    expect(
      within(card(renamed)).getByText(wording.revision(first.revision + 1)),
    ).toBeInTheDocument();
    await publish(renamed);
    await expectAnnounced(wording.published(renamed));
    expect(decisions().map((request) => request.body)).toEqual([
      { revision: first.revision, decision: 'publier' },
      { revision: first.revision + 1, decision: 'publier' },
    ]);
  });
});

describe('US-022 AC5 écran réservé à l’admin', () => {
  it.each(['vendeur', 'acheteur'] as const)(
    'Given un %s connecté, When il ouvre l’écran admin, Then « Accès réservé » sans appel à l’API d’administration',
    async (role) => {
      givenToReview(first);

      await signInAs(accounts[role], adminScreenPath);

      expect(
        await screen.findByRole('heading', { level: 1, name: wording.forbidden }),
      ).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: /^Publier « / })).not.toBeInTheDocument();
      expect(administration().requests).toEqual([]);
    },
  );
});

describe('US-022 AC7 sorties neutralisées sur l’écran admin', () => {
  it('Given un titre contenant du balisage, When l’écran l’affiche, Then il apparaît comme du texte sans créer d’élément ni exécuter de script', async () => {
    givenToReview(
      toReview(1, {
        titre: markupTitle,
        description: '<i>description</i><script>window.__us022 = true</script>',
      }),
    );

    await openAdminScreenWith(1);

    const heading = screen.getByRole('heading', { level: 3, name: markupTitle });
    expect(heading.textContent).toBe(markupTitle);
    expect(heading.children).toHaveLength(0);
    const root = document.getElementById('root');
    expect(root?.querySelectorAll('script, b, i, img[src="x"]')).toHaveLength(0);
    expect(
      screen.getByText('<i>description</i><script>window.__us022 = true</script>'),
    ).toBeInTheDocument();
    expect((window as unknown as { __us022?: boolean }).__us022).toBeUndefined();
  });

  it('Given un titre contenant du balisage, When l’admin ouvre le refus puis publie, Then le titre reste du texte dans le dialogue, les libellés et l’annonce', async () => {
    givenToReview(toReview(1, { titre: markupTitle }));
    await openAdminScreenWith(1);

    const dialog = await openRefusal(markupTitle);
    expect(
      within(dialog).getByText(new RegExp(markupTitle.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))),
    ).toBeInTheDocument();
    await closeRefusal(dialog);
    await publish(markupTitle);

    await expectAnnounced(wording.published(markupTitle));
    expect(document.querySelectorAll('#root script, #root b, #root img[src="x"]')).toHaveLength(0);
    expect((window as unknown as { __us022?: boolean }).__us022).toBeUndefined();
  });
});

describe('US-022 AC8 axe-core sur l’écran admin', () => {
  it(
    'Given la liste des articles à vérifier, When axe-core l’analyse, Then aucune violation serious ou critical',
    async () => {
      givenToReview(first, second);
      await openAdminScreenWith(2);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it(
    'Given le dialogue de refus ouvert avec l’erreur de motif, When axe-core l’analyse, Then aucune violation serious ou critical',
    async () => {
      givenToReview(first);
      await openAdminScreenWith(1);
      const dialog = await openRefusal(first.titre);
      await userEvent.click(within(dialog).getByRole('button', { name: wording.confirm }));
      await within(dialog).findByText(wording.missingReason);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it(
    'Given l’invitation à recharger après un 409, When axe-core l’analyse, Then aucune violation serious ou critical',
    async () => {
      givenToReview(first);
      await openAdminScreenWith(1);
      changeRevision(first.id);
      await publish(first.titre);
      await expectConflictShown();

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it(
    'Given l’état vide, When axe-core l’analyse, Then aucune violation serious ou critical',
    async () => {
      await openAdminScreenWith(0);

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );

  it(
    'Given l’accès réservé pour un vendeur, When axe-core l’analyse, Then aucune violation serious ou critical',
    async () => {
      await signInAs(accounts.vendeur, adminScreenPath);
      await screen.findByRole('heading', { level: 1, name: wording.forbidden });

      expect(await blockingViolations()).toEqual([]);
    },
    axeTimeout,
  );
});
