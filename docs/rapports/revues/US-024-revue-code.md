# US-024 — Revue de code (back)

| Champ | Valeur |
|---|---|
| US | [US-024 — Aucune annonce publiée ni perdue quand le contrôle est indisponible](../../../backlog/US-024-controle-ferme-par-defaut.md) |
| Branche | `feat/US-024-controle-ferme-par-defaut` (base `origin/main`, commits `750f5c6`, `98a513e`) |
| Date | 2026-11-06 |
| Périmètre | `libs/platform/broker/consumer.go`, `libs/platform/outbox/` (purge, rejeu), `services/place-de-marche` (filet de relance, commande `rejouer`, stockage, migration 00007, tests), README de la Place de marché et du Contrôle, rapport de tests ; 16 fichiers |
| Références | [D05](../../decisions/D05-communication-interservice.md) (MC6, MC8, MC9, filet), `infra/canal/broker/definitions/10-topologie.json`, `services/place-de-marche/README.md`, [rapport de tests](../tests/US-024-rapport-tests.md) |
| Verdict | **Fusion après arbitrage** (0 bloquante, 1 majeure, 3 mineures, 4 suggestions) |

## Vérifications exécutées

| Commande | Résultat |
|---|---|
| `gofmt -l .`, `go vet ./...` (place-de-marche, libs/platform, controle) | Aucun écart |
| `golangci-lint run ./...` (place-de-marche, libs/platform) | 0 issue |
| Suite testcontainers | Non relancée : reprise du rapport de tests (221 s, 0 skip, vert) |
| Recherche de commentaires dans le diff Go et SQL | Aucun (seules les annotations goose, requises) |
| Recherche de l'ancien événement `canal.emetteur_refuse` | Plus aucune référence hors `sandbox/XP02` (indépendant) ; aucune règle d'alerte ne le consommait encore |
| Recherche des écritures `etat = 'en_controle'` (contrainte `article_demande_controle_datee`) | Seules `submitQuery` et `reviseQuery` y mènent, toutes deux posent `demande_controle_le` ; pas de régression |
| Recherche de mentions d'outils d'IA dans le diff et les commits | Aucune |

## Points forts

- Mise à l'écart unifiée : `consumer.go` passe l'émetteur inattendu et le refus du traitement par le même `discard`, journalisé seulement après un `Nack` réussi ; le `settle` ne connaît plus que `Ack`/`Reject`, conforme à l'énumération `Disposition` (deux valeurs). Les deux consommateurs (`controle/cmd/controle/main.go:57`, `place-de-marche/.../serve.go:86`) héritent du changement sans modification.
- Filet concurrent-sûr : `resendOverdueQuery` (`postgres.go:62`) verrouille en `FOR UPDATE SKIP LOCKED`, remet la date et écrit la demande dans la même transaction que `requestReview` (`postgres.go:311-330`) ; la demande est reconstruite à la révision courante, donc toujours absorbable par la garde de révision et `resultat_controle_une_issue_par_revision`. Index partiel `article_en_controle_idx` aligné sur le tri du balayage.
- Purge en moindre privilège : `marche_execution` n'a pas `DELETE` sur `outbox` (privilèges par défaut de 00001 : `SELECT, INSERT, UPDATE`) ; fonction `SECURITY DEFINER` sans paramètre, `search_path = public, pg_temp` (pg_temp en dernier, pg_catalog implicite en tête), `REVOKE ... FROM PUBLIC` puis `GRANT EXECUTE` au seul rôle d'exécution, condition `publie_le < ...` qui exclut les lignes non publiées. Index `outbox_publiee_idx` dédié.
- Rejeu conforme à MC9 : seule `publie_le` est écrite (`replay.go:12`), fenêtre semi-ouverte validée avant toute connexion (`NewWindow`), plafonnée à la conservation, arguments surnuméraires refusés (`rejeu.go:60`), trace `outbox.rejeu`.
- Requête I3 du README rejouée telle quelle par les tests (`controle_ferme_test.go:37`), qui distinguent demandes dépassées et relances.
- Docs à jour dans le même lot : README Place de marché (section « Contrôle fermé par défaut », ligne de commande `rejouer`) et README Contrôle (motifs, `delivery-limit`).

## Issues

### Bloquantes

Aucune.

### Majeures

1. `services/place-de-marche/internal/annonces/controle/safeguard.go:43-55` et `internal/annonces/stockage/postgres.go:62-79` — Le filet réémet une demande toutes les 2 min par article `en_controle`, sans plafond ni espacement croissant, et sans regarder si la précédente attend encore dans l'outbox ou dans la file. Conséquences constatées sur la topologie réelle (`10-topologie.json:39` : `max-length` 10 000 en `reject-publish`, durée de vie 24 h vers `.ecartes`) :
   - Contrôle arrêté : la file `controle.demandes` reçoit N copies toutes les 2 min (N articles en attente) ; avec 100 articles elle est pleine en un peu plus de 3 h, après quoi le broker refuse toute publication et les **nouvelles** soumissions restent bloquées dans l'outbox derrière les copies (le relais publie par `cree_le`). Passé 24 h, chaque copie expirée part vers `controle.demandes.ecartes`, dont chaque entrée doit déclencher une alerte (D05, MC8).
   - Demande écartée de façon déterministe (contrat non respecté par la Place de marché, ou résultat écarté côté `pdm.resultats-controle`) : l'article reste `en_controle` et la même demande est réémise puis réécartée indéfiniment. Le « nombre borné d'essais » d'AC3 n'est vrai qu'au niveau d'un message, plus au niveau de l'article ; l'outbox gagne 720 lignes par jour et par article.
   D05 ne fixe pas de plafond, mais il présente le filet comme couverture de la replanification du pod du broker, pas comme relance permanente (« Le filet reste, pour la replanification du pod du broker, pas comme mécanisme principal »). À arbitrer avec Martin ; options simples : espacement croissant par article (colonne `relances`, délai `2 min × 2^relances` plafonné, par exemple à 1 h), ou plafond de relances suivi d'un journal d'alerte dédié et d'un arrêt du filet pour cet article ; dans les deux cas, documenter la règle dans le README et D05. Dev : back.

### Mineures

1. `services/place-de-marche/internal/annonces/controle/safeguard.go:12-13,26` — Délai (2 min) et période (30 s) non injectables : `NewSafeguard` ne prend que le stockage et le journal. Le testeur a dû antidater `demande_controle_le` en base (`controle_ferme_test.go:190`) puis attendre le ticker réel (jusqu'à 30 s par passage). Comme `ResendOverdueReviews` reçoit déjà le délai en paramètre, passer un `SafeguardTiming{OverdueAfter, SweepEvery}` au constructeur (valeurs D05 par défaut dans `serve.go`, lues éventuellement de la configuration) rend le filet testable sans écriture directe en base et réduit la durée de la suite. Dev : back.
2. `libs/platform/outbox/purge.go:12,54` et `internal/migrations/00007_relancer_le_controle_et_purger_l_outbox.sql:21` — Conservation de 7 jours écrite deux fois : `Retention` en Go (plafond du rejeu, champ `conservation` du journal de purge) et `interval '7 days'` dans la fonction SQL qui décide réellement. La fonction sans paramètre est un choix de sécurité à garder ; en revanche le journal `outbox.purge` affirme une valeur que le code Go n'applique pas, et rien ne détecte une divergence. Au choix : exposer la durée côté base (fonction `IMMUTABLE` `conservation_outbox()` lue par la purge et par `NewWindow` au lancement du rejeu), ou ajouter un test qui compare `Retention` à l'intervalle de la fonction (`pg_get_functiondef`) et retirer `conservation` du journal. Dev : back.
3. `libs/platform/outbox/purge.go:12`, `libs/platform/outbox/replay.go:14` — `Retention` et `ErrInvalidWindow` sont exportés mais utilisés uniquement dans le paquet `outbox` (aucune référence ailleurs, tests compris). Les rendre privés, ou s'en servir réellement (`errors.Is(err, outbox.ErrInvalidWindow)` pour distinguer usage et fenêtre dans le code de sortie de `rejouer`). Dev : back.

### Suggestions

1. `services/place-de-marche/cmd/place-de-marche/rejeu.go:32-40` et `comptes.go:23-31` — Même séquence `config.LoadDatabase` → `stockage.Connect` → `defer pool.Close()` dans les deux commandes ponctuelles ; un `withDatabase(ctx, func(*pgxpool.Pool) error) error` les ramènerait à leur seule logique. Dev : back.
2. `libs/platform/broker/consumer.go:14,178-179` — Le nom `file_ecartes` est déduit par convention (`Queue + ".ecartes"`) alors que la destination réelle est fixée par l'IaC (`dead-letter-exchange` + liaison). Exact aujourd'hui pour les trois files ; un champ `DiscardQueue` du `Consumer`, renseigné par la configuration, éviterait qu'un journal mente après une évolution de topologie. Dev : back.
3. `services/place-de-marche/internal/annonces/controle/safeguard.go:30-41` — Le filet attend 30 s avant son premier balayage, alors que la purge passe dès le démarrage (`purge.go:28-30`). Balayer au démarrage rétablirait plus vite les articles en souffrance après un redéploiement (cas de la replanification du broker visé par D05). Dev : back.
4. D05 § « Développement » (MC9) parle de `replay --since` avec une fenêtre par défaut « depuis le début de la session » ; le code livre `rejouer --depuis` obligatoire. Le README fait foi, mais une ligne de complément dans D05 éviterait la question en soutenance. Dev : back (doc).

## Verdict

| Sévérité | Nombre |
|---|---|
| Bloquante | 0 |
| Majeure | 1 |
| Mineure | 3 |
| Suggestion | 4 |

**Fusion après arbitrage.** Aucun bug de transaction, de concurrence ni de droits constaté ; tous les AC sont couverts. La majeure tient à une règle que D05 ne fixe pas (borne ou espacement des relances) : Martin tranche, puis le back corrige ou documente l'acceptation du risque. Les mineures peuvent suivre dans ce lot.
