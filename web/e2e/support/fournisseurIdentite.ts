import type { Page, Route } from '@playwright/test';

export const autoriteE2e = 'https://auth.collector.e2e.localhost/realms/collector';
export const clientIdE2e = 'collector-web';

export type CompteFictif = {
  readonly sujet: string;
  readonly nom: string;
  readonly identifiant: string;
  readonly roles: readonly string[];
};

export const vendeur: CompteFictif = {
  sujet: 'vendeur-e2e',
  nom: 'Alice Vendeur',
  identifiant: 'vendeur.e2e',
  roles: ['vendeur'],
};

export const acheteur: CompteFictif = {
  sujet: 'acheteur-e2e',
  nom: 'Bruno Acheteur',
  identifiant: 'acheteur.e2e',
  roles: ['acheteur'],
};

const dureeJetonSecondes = 900;
const enTetesCors = { 'Access-Control-Allow-Origin': '*' };
const suffixeDecouverte = '/.well-known/openid-configuration';
const pointsDeTerminaison = {
  autorisation: '/protocol/openid-connect/auth',
  jeton: '/protocol/openid-connect/token',
  revocation: '/protocol/openid-connect/revoke',
  finDeSession: '/protocol/openid-connect/logout',
  certificats: '/protocol/openid-connect/certs',
} as const;

function base64Url(valeur: unknown): string {
  return Buffer.from(JSON.stringify(valeur)).toString('base64url');
}

function jwtNonSigne(revendications: Record<string, unknown>): string {
  return `${base64Url({ alg: 'RS256', typ: 'JWT', kid: 'e2e' })}.${base64Url(revendications)}.c2lnbmF0dXJl`;
}

function decouverte() {
  const point = (chemin: string) => `${autoriteE2e}${chemin}`;
  return {
    issuer: autoriteE2e,
    authorization_endpoint: point(pointsDeTerminaison.autorisation),
    token_endpoint: point(pointsDeTerminaison.jeton),
    revocation_endpoint: point(pointsDeTerminaison.revocation),
    end_session_endpoint: point(pointsDeTerminaison.finDeSession),
    jwks_uri: point(pointsDeTerminaison.certificats),
  };
}

function jetons(compte: CompteFictif) {
  const maintenant = Math.floor(Date.now() / 1000);
  const communs = {
    iss: autoriteE2e,
    sub: compte.sujet,
    iat: maintenant,
    exp: maintenant + dureeJetonSecondes,
  };
  return {
    access_token: jwtNonSigne({
      ...communs,
      aud: 'account',
      azp: clientIdE2e,
      realm_access: { roles: [...compte.roles, 'default-roles-collector'] },
    }),
    id_token: jwtNonSigne({
      ...communs,
      aud: clientIdE2e,
      name: compte.nom,
      preferred_username: compte.identifiant,
    }),
    refresh_token: `rafraichissement-${compte.sujet}`,
    token_type: 'Bearer',
    expires_in: dureeJetonSecondes,
    scope: 'openid profile',
  };
}

function redirection(route: Route, cible: string): Promise<void> {
  return route.fulfill({ status: 302, headers: { Location: cible } });
}

function retourAutorisation(url: URL): string {
  const etat = url.searchParams.get('state') ?? '';
  const retour = url.searchParams.get('redirect_uri') ?? '/';
  return `${retour}#code=code-e2e&state=${encodeURIComponent(etat)}`;
}

export class FournisseurIdentite {
  compte: CompteFictif = vendeur;

  async brancher(page: Page): Promise<void> {
    await page.route(`${autoriteE2e}/**`, (route) => this.repondre(route));
  }

  private repondre(route: Route): Promise<void> {
    const requete = route.request();
    const url = new URL(requete.url());
    if (requete.method() === 'OPTIONS') {
      return route.fulfill({
        status: 204,
        headers: {
          ...enTetesCors,
          'Access-Control-Allow-Methods': 'GET, POST',
          'Access-Control-Allow-Headers': '*',
        },
      });
    }
    if (url.pathname.endsWith(suffixeDecouverte)) {
      return route.fulfill({ json: decouverte(), headers: enTetesCors });
    }
    if (url.pathname.endsWith(pointsDeTerminaison.autorisation)) {
      return redirection(route, retourAutorisation(url));
    }
    if (url.pathname.endsWith(pointsDeTerminaison.jeton)) {
      return route.fulfill({ json: jetons(this.compte), headers: enTetesCors });
    }
    if (url.pathname.endsWith(pointsDeTerminaison.revocation)) {
      return route.fulfill({ status: 200, headers: enTetesCors, body: '' });
    }
    if (url.pathname.endsWith(pointsDeTerminaison.finDeSession)) {
      return redirection(route, url.searchParams.get('post_logout_redirect_uri') ?? '/');
    }
    return route.fulfill({ status: 404, headers: enTetesCors, json: { error: 'not_found' } });
  }
}
