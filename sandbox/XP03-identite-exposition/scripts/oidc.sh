# Fonctions OIDC communes aux verifications XP03 (sourcee apres lib.sh).
# Aucun secret ni jeton en argument de commande : configuration curl passee par l'entree standard (-K -).
# shellcheck shell=bash

FICHIER_OTP="$GENERE_DIR/derniere-connexion-otp"

aleatoire() { openssl rand -base64 96 | tr -dc 'A-Za-z0-9' | head -c "$1"; }

urlenc() { jq -rn --arg v "$1" '$v|@uri'; }

curl_pub() {
  # Acces par le proxy local (navigateur ou poste) ; autorite locale de Caddy, jamais -k.
  # --ssl-no-revoke : le curl Schannel de Windows ne peut pas verifier la revocation d'une autorite locale.
  curl -sS --cacert "$(chemin_hote "$CA_LOCALE")" --ssl-no-revoke "$@"
}

curl_interne() {
  # Appel depuis le reseau interne (equivalent du namespace transactions-test) ; configuration curl sur stdin.
  case "$PLATEFORME" in
    compose) docker run --rm -i --network xp03_interne "$CURL_IMAGE" -sS -K - ;;
    k3d|eks) kubectl -n transactions-test exec -i deploy/appelant -c curl -- curl -sS -K - ;;
  esac
}

hote_keycloak_interne() {
  case "$PLATEFORME" in
    compose) printf 'http://keycloak:8080' ;;
    k3d|eks) printf 'http://keycloak.identite-test.svc:8080' ;;
  esac
}
hote_verificateur_interne() {
  case "$PLATEFORME" in
    compose) printf 'http://verificateur:8081' ;;
    k3d|eks) printf 'http://verificateur.pdm-test.svc:8081' ;;
  esac
}

jeton_client() {
  # jeton_client <client> <variable du secret> [portee] : jeton d'acces en client credentials (adresse interne).
  local client="$1" var="$2" portee="${3:-}"
  {
    printf 'url = "%s/realms/collector/protocol/openid-connect/token"\n' "$(hote_keycloak_interne)"
    printf 'user = "%s:%s"\n' "$client" "${!var}"
    printf 'data = "grant_type=client_credentials"\n'
    [[ -n "$portee" ]] && printf 'data = "scope=%s"\n' "$portee"
    true
  } | curl_interne | jq -r '.access_token // empty'
}

appel_interne() {
  # appel_interne <chemin> [jeton] : code HTTP du service de test (port interne 8081).
  local chemin="$1" jeton="${2:-}"
  {
    printf 'url = "%s%s"\n' "$(hote_verificateur_interne)" "$chemin"
    printf 'output = "/dev/null"\n'
    printf 'write-out = "%%{http_code}"\n'
    printf 'max-time = 10\n'
    [[ -n "$jeton" ]] && printf 'header = "Authorization: Bearer %s"\n' "$jeton"
    true
  } | curl_interne
}

totp() {
  # Code TOTP (RFC 6238, SHA-1, 6 chiffres, 30 s) de la graine de session (secret brut de Keycloak).
  local cle_hex compteur hmac decalage
  cle_hex="$(printf '%s' "$GRAINE_OTP" | xxd -p | tr -d '\n')"
  compteur="$(printf '%016x' $(( $(date +%s) / 30 )))"
  hmac="$(printf '%s' "$compteur" | xxd -r -p | openssl dgst -sha1 -mac HMAC -macopt "hexkey:$cle_hex" -binary | xxd -p | tr -d '\n')"
  decalage=$(( 16#${hmac:39:1} ))
  printf '%06d' $(( (16#${hmac:decalage*2:8} & 16#7fffffff) % 1000000 ))
}

action_formulaire() { grep -o 'action="[^"]*"' "$1" | head -1 | sed -e 's/^action="//' -e 's/"$//' -e 's/&amp;/\&/g'; }

connexion_navigateur() {
  # connexion_navigateur <utilisateur> <variable du mot de passe> [portee supplementaire]
  # Flux code d'autorisation + PKCE S256 du client public collector-web, par le proxy (jamais le
  # flux "mot de passe"). Ecrit oui|non dans $FICHIER_OTP (OTP demande) ; renvoie le jeton d'acces.
  local u="$1" var="$2" portee="openid${3:+ $3}" verif defi jar page entetes action lieu code cred essai
  verif="$(aleatoire 64)"
  defi="$(printf '%s' "$verif" | openssl dgst -sha256 -binary | base64 | tr '+/' '-_' | tr -d '=\n')"
  jar="$(mktemp "$GENERE_DIR/jar.XXXXXX")"; page="$(mktemp "$GENERE_DIR/page.XXXXXX")"; entetes="$(mktemp "$GENERE_DIR/ent.XXXXXX")"
  curl_pub -c "$(chemin_hote "$jar")" -b "$(chemin_hote "$jar")" -o "$(chemin_hote "$page")" \
    "$URL_AUTH/realms/collector/protocol/openid-connect/auth?client_id=collector-web&response_type=code&scope=$(urlenc "$portee")&redirect_uri=$(urlenc "$URL_FRONT/rappel")&state=xp03&code_challenge=$defi&code_challenge_method=S256"
  action="$(action_formulaire "$page")"
  [[ -n "$action" ]] || { rm -f "$jar" "$page" "$entetes"; echec "formulaire de connexion introuvable"; }
  {
    printf 'url = "%s"\n' "$action"
    printf 'data-urlencode = "username=%s"\n' "$u"
    printf 'data-urlencode = "password=%s"\n' "${!var}"
  } | curl_pub -K - -c "$(chemin_hote "$jar")" -b "$(chemin_hote "$jar")" -o "$(chemin_hote "$page")" -D "$(chemin_hote "$entetes")"
  echo non >"$FICHIER_OTP"
  if grep -q 'name="otp"' "$page"; then
    echo oui >"$FICHIER_OTP"
    # Deux essais au plus : un code calcule en fin de periode de 30 s peut arriver perime
    # (decalage d'horloge entre le poste et la machine virtuelle de Docker).
    for essai in 1 2; do
      action="$(action_formulaire "$page")"
      cred="$(grep -o 'name="selectedCredentialId" value="[^"]*"' "$page" | head -1 | sed -e 's/.*value="//' -e 's/"$//')"
      {
        printf 'url = "%s"\n' "$action"
        printf 'data-urlencode = "otp=%s"\n' "$(totp)"
        [[ -n "$cred" ]] && printf 'data-urlencode = "selectedCredentialId=%s"\n' "$cred"
        true
      } | curl_pub -K - -c "$(chemin_hote "$jar")" -b "$(chemin_hote "$jar")" -o "$(chemin_hote "$page")" -D "$(chemin_hote "$entetes")"
      grep -qi '^location:' "$entetes" && break
      [[ $essai -eq 1 ]] && sleep $(( 31 - $(date +%s) % 30 ))
    done
  fi
  lieu="$(grep -i '^location:' "$entetes" | tail -1 | tr -d '\r' | cut -d' ' -f2-)"
  code="$(sed -n 's/.*[?&]code=\([^&]*\).*/\1/p' <<<"$lieu")"
  rm -f "$jar" "$page" "$entetes"
  [[ -n "$code" ]] || { journal "connexion de $u : aucun code (redirection : ${lieu%%\?*})"; return 1; }
  {
    printf 'url = "%s/realms/collector/protocol/openid-connect/token"\n' "$URL_AUTH"
    printf 'data = "grant_type=authorization_code"\n'
    printf 'data = "client_id=collector-web"\n'
    printf 'data-urlencode = "code=%s"\n' "$code"
    printf 'data-urlencode = "redirect_uri=%s"\n' "$URL_FRONT/rappel"
    printf 'data = "code_verifier=%s"\n' "$verif"
  } | curl_pub -K - | jq -r '.access_token // empty'
}

jeton_admin_master() {
  # Jeton du compte d'administration initial (realm master, admin-cli), adresse interne seulement :
  # la console et l'API d'administration ne sont jamais routees par le proxy.
  {
    printf 'url = "%s/realms/master/protocol/openid-connect/token"\n' "$(hote_keycloak_interne)"
    printf 'data = "grant_type=password"\n'
    printf 'data = "client_id=admin-cli"\n'
    printf 'data = "username=admin-temporaire"\n'
    printf 'data-urlencode = "password=%s"\n' "$MDP_KC_BOOTSTRAP"
  } | curl_interne | jq -r '.access_token // empty'
}

api_admin() {
  # api_admin <jeton> <chemin sous /admin/realms/collector>
  {
    printf 'url = "%s/admin/realms/collector%s"\n' "$(hote_keycloak_interne)" "$2"
    printf 'header = "Authorization: Bearer %s"\n' "$1"
  } | curl_interne
}
