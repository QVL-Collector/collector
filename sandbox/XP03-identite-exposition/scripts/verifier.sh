#!/usr/bin/env bash
# Verifications locales de XP03 (protocole § B) ; resultats dans le CSV et docs/mesures/XP03-identite-exposition/.
# Usage : PLATEFORME=compose|k3d scripts/verifier.sh <verification>
#   k2      demarrage sur base neuve avec import du realm, 2 mesures (compose : pile recreee)
#   xe4     liste blanche du proxy (XE4 + ME8), chemins figes ci-dessous            (compose)
#   k3      six cas contre "reserver" + route d'acheteur (MA5) ; attend l'expiration d'un jeton (~5,5 min)
#   ma12    OTP demande a l'admin de test, pas au vendeur de test
#   ma      parametres du realm et refus de flux (MA1, MA2, MA3, MA5, MA9, MA11, MA13)
#   trivy   analyse de l'image Keycloak (MA14)
#   tout    xe4, ma12, ma, k3 (dans cet ordre)
source "$(dirname "${BASH_SOURCE[0]}")/lib.sh"
source "$XP_DIR/scripts/oidc.sh"
charger_env

if [[ "$PLATEFORME" == k3d && "${1:-}" =~ ^(k3|ma12|ma|tout)$ ]]; then
  # Acces du poste a Keycloak par port-forward (equivalent local du proxy, emetteur de debogage).
  kubectl -n identite-test port-forward svc/keycloak "$PORT_FORWARD_K3D:8080" >/dev/null 2>&1 &
  pf=$!
  trap 'kill "$pf" 2>/dev/null || true' EXIT
  sleep 3
fi

TOTAL=0; OK=0
resultat() {
  # resultat <critere> <etape> <obtenu> <attendu> <commentaire>
  local atteint; atteint="$(oui_non "$3" "$4")"
  TOTAL=$((TOTAL + 1)); [[ "$atteint" == oui ]] && OK=$((OK + 1))
  csv "$1" "$2" "$3" "resultat" "$4" "$atteint" "$5"
  printf '%-6s %-55s obtenu=%-8s attendu=%-8s %s\n' "$1" "$2" "$3" "$4" "$atteint" >&2
}

# Liste figee des chemins (protocole XP03 § C3, completee des traversees vers les chemins autorises).
CHEMINS_200=(
  "/realms/collector/.well-known/openid-configuration"
  "/realms/collector/protocol/openid-connect/certs"
)
CHEMINS_404=(
  "/admin/" "/admin/master/console/" "/realms/master/" "/realms/master/protocol/openid-connect/certs"
  "/realms/collector/account" "/realms/collector/account/"
  "/realms/collector/../master/" "/realms/collector/%2e%2e/master/" "/%2e%2e/admin/" "/%2E%2E/admin/"
  "/realms%2fmaster/" "/realms%2Fmaster/" "/ADMIN/" "/Realms/Master/" "/resources/../admin/"
  "/realms/collector/protocol/openid-connect/../../../master/"
  "/realms/collector/protocol/openid-connect/%2e%2e/%2e%2e/%2e%2e/master/"
  "/realms/collector/protocol/openid-connect%2f..%2f..%2f..%2fmaster/"
  "/resources/..%2f..%2fadmin/" "/resources/%2e%2e/%2e%2e/admin/master/console/"
  "/realms/collector/login-actions/../../master/" "/realms/collector/protocol/openid-connect\\..\\..\\..\\master/"
  "/realms/collector/clients-registrations/openid-connect" "/realms/collector/broker/" "/metrics" "/health/ready"
)

xe4() {
  [[ "$PLATEFORME" == compose ]] || echec "xe4 : liste blanche du proxy local (compose) ; sur EKS, celle de l'ALB"
  local sortie p code
  csv_init
  sortie="$SORTIES_DIR/$(horodatage)-$PLATEFORME-xe4.txt"
  for p in "${CHEMINS_200[@]}"; do
    code="$(curl_pub --path-as-is -o "$POUBELLE" -w '%{http_code}' "$URL_AUTH$p" || true)"
    echo "$p $code" >>"$sortie"; resultat XE4 "$p" "$code" 200 "liste blanche"
  done
  code="$(curl_pub -o "$POUBELLE" -w '%{http_code}' "$URL_AUTH/realms/collector/protocol/openid-connect/auth?client_id=collector-web&response_type=code&scope=openid&redirect_uri=$(urlenc "$URL_FRONT/rappel")&code_challenge=$(aleatoire 43)&code_challenge_method=S256" || true)"
  echo "page-de-connexion $code" >>"$sortie"; resultat XE4 "page de connexion (auth, PKCE)" "$code" 200 "liste blanche"
  for p in "${CHEMINS_404[@]}"; do
    code="$(curl_pub --path-as-is -o "$POUBELLE" -w '%{http_code}' "$URL_AUTH$p" || true)"
    echo "$p $code" >>"$sortie"; resultat ME8 "$p" "$code" 404 "chemin interdit"
  done
  code="$(curl_pub --path-as-is -o "$POUBELLE" -w '%{http_code}' "$URL_FRONT/reserver" || true)"
  echo "front:/reserver $code" >>"$sortie"; resultat XE4 "front /reserver" "$code" 404 "reserver jamais route"
  # Port de gestion 9000 : jamais publie. Local : connexion refusee ; EKS : delai expire.
  code="$(curl_pub --max-time 5 -o "$POUBELLE" -w '%{http_code}' "https://$HOTE_AUTH:9000/health" 2>/dev/null || true)"
  echo "port-9000 ${code:-000}" >>"$sortie"; resultat XE4 "port 9000" "${code:-000}" 000 "refus ou delai expire"
  code="$(curl -s --max-time 5 -o "$POUBELLE" -w '%{http_code}' "http://$HOTE_FRONT:$PORT_HTTP/" || true)"
  echo "http-front $code" >>"$sortie"; resultat XE3 "http -> https (indicatif local)" "$code" 301 "redirection 80"
}

k3() {
  csv_init
  local expire exp_ts acheteur autre_pdm autre_inter transactions attente
  journal "K3 : jeton de transactions destine a expirer (obtenu maintenant)"
  expire="$(jeton_client transactions SECRET_TRANSACTIONS)"
  [[ -n "$expire" ]] || echec "jeton de transactions non obtenu"
  exp_ts="$(resume_jeton <<<"$expire" | jq -r .exp)"

  acheteur="$(connexion_navigateur acheteur-test MDP_ACHETEUR)"
  autre_pdm="$(jeton_client xp03-autre SECRET_AUTRE aud-place-de-marche)"
  autre_inter="$(jeton_client xp03-autre SECRET_AUTRE aud-interactions)"
  transactions="$(jeton_client transactions SECRET_TRANSACTIONS)"
  {
    echo "acheteur     $(resume_jeton <<<"$acheteur")"
    echo "autre-pdm    $(resume_jeton <<<"$autre_pdm")"
    echo "autre-inter  $(resume_jeton <<<"$autre_inter")"
    echo "transactions $(resume_jeton <<<"$transactions")"
  } | tee "$SORTIES_DIR/$(horodatage)-$PLATEFORME-k3-charges-utiles.txt" >&2

  resultat K3 "sans jeton" "$(appel_interne /reserver)" 401 "reserver"
  resultat K3 "jeton d'utilisateur (acheteur, PKCE)" "$(appel_interne /reserver "$acheteur")" 401 "reserver"
  resultat K3 "jeton d'un autre client (xp03-autre, aud place-de-marche)" "$(appel_interne /reserver "$autre_pdm")" 401 "reserver"
  resultat K3 "jeton d'une autre audience (xp03-autre, aud interactions)" "$(appel_interne /reserver "$autre_inter")" 401 "reserver"
  resultat K3 "jeton de transactions" "$(appel_interne /reserver "$transactions")" 200 "reserver"
  resultat MA5 "jeton de transactions sur route d'acheteur" "$(appel_interne /achat "$transactions")" 401 "achat"
  resultat MA5 "temoin : jeton d'acheteur sur route d'acheteur" "$(appel_interne /achat "$acheteur")" 200 "achat"
  resultat MA3 "portee reserver absente du jeton d'acheteur" \
    "$(resume_jeton <<<"$acheteur" | jq -r '(.scope // "") | split(" ") | index("reserver") | if . == null then "absente" else "presente" end')" absente "collector-web"

  attente=$(( exp_ts + 35 - $(date +%s) ))
  if (( attente > 0 )); then journal "K3 : attente de l'expiration du jeton ($attente s, tolerance d'horloge 30 s)"; sleep "$attente"; fi
  resultat K3 "jeton expire (transactions, > 5 min)" "$(appel_interne /reserver "$expire")" 401 "reserver"
}

ma12() {
  csv_init
  local j
  j="$(connexion_navigateur admin-test MDP_ADMIN_TEST)" || true
  resultat MA12 "OTP demande a l'admin de test" "$(cat "$FICHIER_OTP")" oui "flux conditionnel role admin"
  resultat MA12 "connexion admin aboutie avec OTP" "$([[ -n "$j" ]] && echo oui || echo non)" oui "code TOTP de la graine de session"
  resultat MA12 "roles du jeton admin" "$(resume_jeton <<<"$j" | jq -r '.roles | index("admin") != null')" true "admin present"
  j="$(connexion_navigateur vendeur-test MDP_VENDEUR)" || true
  resultat MA12 "OTP demande au vendeur de test" "$(cat "$FICHIER_OTP")" non "vendeur sans OTP"
  resultat MA12 "connexion vendeur aboutie" "$([[ -n "$j" ]] && echo oui || echo non)" oui "sans OTP"
}

ma() {
  csv_init
  local adm realm web tr comptes roles_sa code page msg1 msg2
  adm="$(jeton_admin_master)"; [[ -n "$adm" ]] || echec "jeton d'administration non obtenu"
  realm="$(api_admin "$adm" "")"
  web="$(api_admin "$adm" "/clients?clientId=collector-web" | jq '.[0]')"
  tr="$(api_admin "$adm" "/clients?clientId=transactions" | jq '.[0]')"
  api_admin "$adm" "" | jq '{accessTokenLifespan, ssoSessionIdleTimeout, ssoSessionMaxLifespan, revokeRefreshToken,
    refreshTokenMaxReuse, bruteForceProtected, permanentLockout, failureFactor, passwordPolicy, browserFlow}' \
    >"$SORTIES_DIR/$(horodatage)-$PLATEFORME-realm-parametres.json"

  # MA1 : realm versionne sans secret (toutes les valeurs sensibles sont des emplacements ${...}).
  comptes="$(jq '[.. | objects | to_entries[] | select(.key == "secret" or .key == "value")
      | select((.value | type) == "string") | select(.value | test("^\\$\\{[A-Z_]+\\}$") | not)] | length' "$(chemin_hote "$XP_DIR/realm/collector-realm.json")")"
  resultat MA1 "valeurs sensibles en clair dans le realm versionne" "$comptes" 0 "secret, value"
  resultat MA1 "URI de redirection a joker (tous clients)" \
    "$(api_admin "$adm" "/clients" | jq '[.[] | select(.clientId | test("^(collector-web|transactions|xp03-autre)$")) | .redirectUris[]?, .webOrigins[]? | select(test("\\*"))] | length')" 0 "redirectUris, webOrigins"
  resultat MA1 "flux implicite ou mot de passe actif (clients du realm)" \
    "$(api_admin "$adm" "/clients" | jq '[.[] | select(.clientId | test("^(collector-web|transactions|xp03-autre)$")) | select(.implicitFlowEnabled or .directAccessGrantsEnabled)] | length')" 0 "MA1"
  resultat MA1 "Full scope allowed actif" \
    "$(api_admin "$adm" "/clients" | jq '[.[] | select(.clientId | test("^(collector-web|transactions|xp03-autre)$")) | select(.fullScopeAllowed)] | length')" 0 "MA1"

  # MA2 : collector-web public, PKCE S256, flux standard seul ; transactions : comptes de service seuls.
  resultat MA2 "collector-web public + PKCE S256 + flux standard seul" \
    "$(jq -r '[.publicClient, .attributes["pkce.code.challenge.method"], .standardFlowEnabled, .serviceAccountsEnabled] | map(tostring) | join(",")' <<<"$web")" "true,S256,true,false" "MA2"
  resultat MA2 "transactions confidentiel, comptes de service seuls" \
    "$(jq -r '[.publicClient, .standardFlowEnabled, .serviceAccountsEnabled] | map(tostring) | join(",")' <<<"$tr")" "false,false,true" "MA2"
  code="$( { printf 'url = "%s/realms/collector/protocol/openid-connect/token"\n' "$URL_AUTH"
             printf 'data = "grant_type=password"\ndata = "client_id=collector-web"\ndata = "username=acheteur-test"\n'
             printf 'data-urlencode = "password=%s"\n' "$MDP_ACHETEUR"; } | curl_pub -K - | jq -r '.error // "jeton"')"
  resultat MA2 "flux mot de passe sur collector-web" "$code" unauthorized_client "refuse"
  page="$(curl_pub -o "$POUBELLE" -w '%{http_code} %{redirect_url}' "$URL_AUTH/realms/collector/protocol/openid-connect/auth?client_id=collector-web&response_type=token&scope=openid&redirect_uri=$(urlenc "$URL_FRONT/rappel")&nonce=x")"
  resultat MA2 "flux implicite sur collector-web" "$(grep -o 'error=[a-z_]*' <<<"$page" | head -1 | cut -d= -f2)" unauthorized_client "refuse"
  page="$(curl_pub -o "$POUBELLE" -w '%{http_code} %{redirect_url}' "$URL_AUTH/realms/collector/protocol/openid-connect/auth?client_id=collector-web&response_type=code&scope=openid&redirect_uri=$(urlenc "$URL_FRONT/rappel")")"
  resultat MA2 "code d'autorisation sans PKCE" "$(grep -o 'error=[a-z_]*' <<<"$page" | head -1 | cut -d= -f2)" invalid_request "refuse"
  code="$(curl_pub -o "$POUBELLE" -w '%{http_code}' "$URL_AUTH/realms/collector/protocol/openid-connect/auth?client_id=collector-web&response_type=code&scope=openid&redirect_uri=$(urlenc "https://attaquant.invalid/rappel")&code_challenge=$(aleatoire 43)&code_challenge_method=S256")"
  resultat MA2 "URI de redirection etrangere" "$code" 400 "refuse sans redirection"

  # MA3 : la portee reserver n'est rattachee qu'a transactions.
  resultat MA3 "reserver rattachee a collector-web" \
    "$(api_admin "$adm" "/clients/$(jq -r .id <<<"$web")/default-client-scopes" | jq -r '[.[].name] | index("reserver") != null'),$(api_admin "$adm" "/clients/$(jq -r .id <<<"$web")/optional-client-scopes" | jq -r '[.[].name] | index("reserver") != null')" "false,false" "par defaut, optionnelle"

  # MA5 : compte de service de transactions sans role de realm dans son jeton.
  roles_sa="$(api_admin "$adm" "/users/$(api_admin "$adm" "/clients/$(jq -r .id <<<"$tr")/service-account-user" | jq -r .id)/role-mappings/realm/composite" | jq -r '[.[].name] | sort | join(" ")')"
  echo "roles effectifs de service-account-transactions : ${roles_sa:-aucun}" >&2
  resultat MA5 "roles de realm dans le jeton de transactions" \
    "$(jeton_client transactions SECRET_TRANSACTIONS | resume_jeton | jq '.roles | length')" 0 "fullScopeAllowed=false ; roles effectifs du compte : ${roles_sa:-aucun}"

  # MA9 : role admin du realm sans droit sur realm-management ; compte initial temporaire.
  resultat MA9 "roles de client composites du role admin" \
    "$(api_admin "$adm" "/roles/admin/composites" | jq '[.[] | select(.clientRole)] | length')" 0 "realm-management"
  # MA11 : durees et rotation.
  resultat MA11 "jeton d'acces / session inactive / session max (s)" \
    "$(jq -r '[.accessTokenLifespan, .ssoSessionIdleTimeout, .ssoSessionMaxLifespan] | map(tostring) | join("/")' <<<"$realm")" "300/1800/28800" "realm"
  resultat MA11 "rotation du jeton de rafraichissement (revoke, reuse)" \
    "$(jq -r '[.revokeRefreshToken, .refreshTokenMaxReuse] | map(tostring) | join(",")' <<<"$realm")" "true,0" "realm"
  # MA13 : force brute sans verrouillage permanent, politique de mot de passe, message identique.
  resultat MA13 "force brute active, verrouillage permanent inactif" \
    "$(jq -r '[.bruteForceProtected, .permanentLockout] | map(tostring) | join(",")' <<<"$realm")" "true,false" "realm"
  resultat MA13 "politique : longueur et liste interdite" \
    "$(jq -r '.passwordPolicy | (test("length\\(12\\)") and test("passwordBlacklist")) | tostring' <<<"$realm")" true "realm"
  msg1="$(message_echec acheteur-test)"; msg2="$(message_echec "inconnu-$(aleatoire 6)")"
  resultat MA13 "message identique compte existant / inexistant" "$([[ -n "$msg1" && "$msg1" == "$msg2" ]] && echo oui || echo non)" oui "\"$msg1\""
}

message_echec() {
  # Message d'erreur affiche apres un mot de passe faux pour l'utilisateur donne (une seule tentative).
  local jar page action
  jar="$(mktemp "$GENERE_DIR/jar.XXXXXX")"; page="$(mktemp "$GENERE_DIR/page.XXXXXX")"
  curl_pub -c "$(chemin_hote "$jar")" -b "$(chemin_hote "$jar")" -o "$(chemin_hote "$page")" \
    "$URL_AUTH/realms/collector/protocol/openid-connect/auth?client_id=collector-web&response_type=code&scope=openid&redirect_uri=$(urlenc "$URL_FRONT/rappel")&code_challenge=$(aleatoire 43)&code_challenge_method=S256"
  action="$(action_formulaire "$page")"
  { printf 'url = "%s"\n' "$action"; printf 'data-urlencode = "username=%s"\n' "$1"; printf 'data-urlencode = "password=%s"\n' "$(aleatoire 20)"; } |
    curl_pub -K - -c "$(chemin_hote "$jar")" -b "$(chemin_hote "$jar")" -o "$(chemin_hote "$page")"
  grep -A1 'kc-feedback-text' "$page" | sed -n '2p' | tr -d "[:cntrl:]" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//'
  rm -f "$jar" "$page"
}

k2() {
  [[ "$PLATEFORME" == compose ]] || echec "k2 compose seulement ici ; k3d : scripts/k3d.sh k2"
  csv_init
  local n debut fin
  for n in 1 2; do
    "$XP_DIR/scripts/compose.sh" dc rm -sfv keycloak postgres >/dev/null 2>&1
    "$XP_DIR/scripts/compose.sh" dc up -d --wait postgres >/dev/null 2>&1
    debut="$(ms_maintenant)"
    "$XP_DIR/scripts/compose.sh" dc up -d --no-deps keycloak >/dev/null 2>&1
    until docker exec xp03-keycloak-1 bash -c 'exec 3<>/dev/tcp/127.0.0.1/9000 && printf "GET /health/ready HTTP/1.0\r\n\r\n" >&3 && read -r l <&3 && [[ "$l" == *" 200 "* ]]' 2>/dev/null; do
      (( ($(ms_maintenant) - debut) / 1000 < 300 )) || echec "Keycloak non pret en 300 s"
      sleep 1
    done
    fin="$(ms_maintenant)"
    local d=$(( (fin - debut) / 1000 ))
    csv K2 "mesure $n" "$d" s "<=180" "$([[ $d -le 180 ]] && echo oui || echo non)" "creation du conteneur -> /health/ready 200, base neuve, import du realm (indicatif local)"
    journal "K2 mesure $n : $d s"
  done
}

trivy() {
  csv_init
  local f="$SORTIES_DIR/$(horodatage)-trivy-keycloak.txt" n rc=0
  # Cache des bases de Trivy dans un volume nomme (supprime par compose.sh down).
  docker run --rm -v /var/run/docker.sock:/var/run/docker.sock -v xp03-trivy-cache:/root/.cache "$TRIVY_IMAGE" image --quiet     --severity CRITICAL,HIGH --ignore-unfixed --format table "$KEYCLOAK_IMAGE" >"$f" 2>&1 || rc=$?
  if [[ $rc -ne 0 ]] || grep -q FATAL "$f"; then
    resultat MA14 "analyse Trivy" "erreur" "0" "analyse non aboutie, detail : $(basename "$f")"
    return
  fi
  n="$(grep -Eo 'Total: [0-9]+' "$f" | awk '{s+=$2} END {print s+0}')"
  resultat MA14 "vulnerabilites critiques ou elevees corrigeables" "$n" 0 "Trivy, detail : $(basename "$f")"
}

case "${1:-}" in
  k2) k2 ;;
  xe4) xe4 ;;
  k3) k3 ;;
  ma12) ma12 ;;
  ma) ma ;;
  trivy) trivy ;;
  tout) xe4; ma12; ma; k3 ;;
  *) echec "usage : $0 k2|xe4|k3|ma12|ma|trivy|tout" ;;
esac
journal "$1 : $OK / $TOTAL conformes"
[[ $OK -eq $TOTAL ]]
