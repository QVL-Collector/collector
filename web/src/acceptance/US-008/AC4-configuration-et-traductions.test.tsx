import { screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { en } from '../../i18n/locales/en';
import { fr } from '../../i18n/locales/fr';
import {
  authorizationEndpoint,
  buyer,
  defaultConfig,
  identityProvider,
  openCollector,
  returnFromIdentityProvider,
  sellerA,
  serveConfig,
  signInAs,
  switchLanguage,
  waitForNavigationTo,
  press,
} from './identityProviderHarness';

const otherAuthority = 'https://auth.autre-domaine.example/realms/collector';

function withOidc(oidc: unknown): unknown {
  return { ...defaultConfig, oidc };
}

async function clickSignIn(name: string): Promise<void> {
  const signIn = await screen.findByRole('button', { name });
  await press(signIn);
}

const invalidConfigurations: readonly (readonly [string, unknown])[] = [
  ['sans section oidc', { marketplaceApiUrl: '/marche', catalogue: defaultConfig.catalogue }],
  [
    'autorité en HTTP hors localhost',
    withOidc({
      ...defaultConfig.oidc,
      authority: 'http://auth.collector.qvl-project.com/realms/collector',
    }),
  ],
  [
    'autorité avec requête',
    withOidc({ ...defaultConfig.oidc, authority: `${defaultConfig.oidc.authority}?x=1` }),
  ],
  [
    'autorité avec fragment',
    withOidc({ ...defaultConfig.oidc, authority: `${defaultConfig.oidc.authority}#x` }),
  ],
  ['autorité qui n’est pas une URL', withOidc({ ...defaultConfig.oidc, authority: 'pas-une-url' })],
  ['client absent', withOidc({ authority: defaultConfig.oidc.authority, scope: 'openid profile' })],
  ['client vide', withOidc({ ...defaultConfig.oidc, clientId: '' })],
  ['portée sans openid', withOidc({ ...defaultConfig.oidc, scope: 'profile email' })],
  ['configuration non objet', 'oidc'],
];

describe('US-008-AC4 configuration lue à l’exécution', () => {
  it('Given le front démarre, When il se lance, Then il lit /config.json à l’exécution', async () => {
    await openCollector('/');

    await screen.findByRole('button', { name: 'Se connecter' });
    expect(
      identityProvider().requests.some((request) => request.url.pathname === '/config.json'),
    ).toBe(true);
  });

  it('Given un config.json désignant un autre serveur d’autorisation, When le visiteur se connecte, Then il est redirigé vers ce serveur sans reconstruction', async () => {
    serveConfig(withOidc({ ...defaultConfig.oidc, authority: otherAuthority }));
    await openCollector('/');

    await clickSignIn('Se connecter');
    const authorize = await waitForNavigationTo(`${otherAuthority}/protocol/openid-connect/auth`);

    expect(authorize.origin).toBe('https://auth.autre-domaine.example');
  });

  it.each(invalidConfigurations)(
    'Given une configuration OIDC invalide (%s), When le front démarre, Then il affiche l’échec de démarrage et ne propose pas la connexion',
    async (_, config) => {
      serveConfig(config);

      await openCollector('/');

      expect(await screen.findByText(fr.app.bootstrapFailure.title)).toBeInTheDocument();
      expect(screen.queryByRole('button', { name: fr.auth.signIn })).not.toBeInTheDocument();
    },
  );

  it('Given config.json introuvable, When le front démarre, Then il affiche l’échec de démarrage', async () => {
    serveConfig(undefined);

    await openCollector('/');

    expect(await screen.findByText(fr.app.bootstrapFailure.title)).toBeInTheDocument();
  });
});

describe('US-008-AC4 textes issus des fichiers de traduction, français par défaut', () => {
  it('Given un visiteur, When le site s’affiche sans choix de langue, Then le bouton de connexion est en français', async () => {
    await openCollector('/');

    expect(await screen.findByRole('button', { name: fr.auth.signIn })).toBeInTheDocument();
    expect(document.documentElement).toHaveAttribute('lang', 'fr');
  });

  it('Given le vendeur A connecté, When l’en-tête s’affiche en français, Then rôle et déconnexion viennent du fichier de traduction français', async () => {
    await signInAs(sellerA);

    const header = screen.getByRole('banner');
    expect(await within(header).findByText(fr.auth.roles.vendeur)).toBeInTheDocument();
    expect(within(header).getByRole('button', { name: fr.auth.signOut })).toBeInTheDocument();
  });

  it('Given le vendeur A connecté, When il choisit English, Then rôle et déconnexion passent en anglais sans rechargement', async () => {
    await signInAs(sellerA);
    await screen.findByText('Alice Vendeur');

    await switchLanguage('en');

    const header = screen.getByRole('banner');
    expect(await within(header).findByText(en.auth.roles.vendeur)).toBeInTheDocument();
    expect(within(header).getByRole('button', { name: en.auth.signOut })).toBeInTheDocument();
    expect(screen.getByText('Alice Vendeur')).toBeInTheDocument();
  });

  it('Given la langue anglaise, When un visiteur ouvre le site, Then le bouton de connexion s’intitule « Sign in »', async () => {
    await openCollector('/');
    await screen.findByRole('button', { name: fr.auth.signIn });

    await switchLanguage('en');

    expect(await screen.findByRole('button', { name: en.auth.signIn })).toBeInTheDocument();
    expect(document.documentElement).toHaveAttribute('lang', 'en');
  });

  it('Given la langue anglaise et un acheteur, When il ouvre /vendeur, Then la page d’accès refusé est en anglais', async () => {
    await signInAs(buyer, '/vendeur');
    await screen.findByRole('heading', { level: 1, name: fr.accessDenied.title });

    await switchLanguage('en');

    expect(
      await screen.findByRole('heading', { level: 1, name: en.accessDenied.title }),
    ).toBeInTheDocument();
  });

  it('Given la langue anglaise, When la connexion échoue au retour, Then le message d’échec est en anglais', async () => {
    await openCollector('/');
    await clickSignIn(fr.auth.signIn);
    await waitForNavigationTo(authorizationEndpoint());
    await returnFromIdentityProvider(
      (state) => `error=access_denied&state=${encodeURIComponent(state)}`,
    );
    await screen.findByText(fr.auth.callback.title);

    await switchLanguage('en');

    expect(await screen.findByText(en.auth.callback.title)).toBeInTheDocument();
  });
});
