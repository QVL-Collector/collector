# Sprint 2 — Mise en vente contrôlée complète, charge et autoscaling

Processus et cérémonies : [developpement.md](../../docs/processus/developpement.md). Clôture : [PILOTAGE.md](../../docs/PILOTAGE.md) § 3.4. Backlog : [BACKLOG.md](../BACKLOG.md).

| Champ | Valeur |
|---|---|
| Jalon | J5 |
| Dates | Prévues : 2026-11-10 → 2026-11-14. Observées (premier et dernier merge du sprint) : 2026-10-28 → 2026-11-06 |
| Statut | Clos sur le périmètre local ; US AWS reportées au sprint 3 (bilan du 2026-11-20) |

## Objectif

À la fin du sprint, la mise en vente contrôlée est complète (quatre règles, photo, revalidation de toute modification, écran « à vérifier », français et anglais accessibles) et la charge est démontrée sur AWS : montée en charge du catalogue et perte d'une instance sans interruption (C3, C5, C6, avec C7 et C8). Contenu tiré de [D02](../../docs/decisions/D02-decoupage-applicatif.md), « Découpage indicatif en sprints », ligne S2.

## US engagées

Ordre d'engagement = rang du backlog. Statuts au 2026-11-20.

| US | Titre | Fonction(s) | Engagement | Statut | Dépend de | PR |
|---|---|---|---|---|---|---|
| [US-015](../US-015-metriques-tableaux-de-bord.md) | Métriques et tableaux de bord | DevOps, Back | Ferme | À faire, reportée au sprint 3 | US-007, US-013 | — |
| [US-016](../US-016-montee-en-charge-catalogue.md) | Absorber un pic de visiteurs sur le catalogue | DevOps, Test | Ferme | À faire, reportée au sprint 3 | US-010, US-015 | — |
| [US-017](../US-017-perte-instance-sous-charge.md) | Rester disponible quand une instance tombe sous charge | DevOps | Ferme | À faire, reportée au sprint 3 | US-016 | — |
| [US-018](../US-018-filtrage-coordonnees.md) | Refuser les annonces avec coordonnées personnelles | Back | Ferme | Livrée | US-013 | [#22](https://github.com/QVL-Collector/collector/pull/22) |
| [US-019](../US-019-regles-prix-statut-vendeur.md) | Mettre à vérifier les prix incohérents et les vendeurs à risque | Back | Ferme | Livrée | US-013 | [#26](https://github.com/QVL-Collector/collector/pull/26), [#32](https://github.com/QVL-Collector/collector/pull/32) |
| [US-021](../US-021-revalidation-apres-modification.md) | Recontrôler toute modification d'un article | Back, Front | Ferme | Livrée | US-013 | [#27](https://github.com/QVL-Collector/collector/pull/27) |
| [US-020](../US-020-photo-securisee.md) | Ajouter une photo à un article, en toute sécurité | Back, Front, DevOps | Ferme | À faire, reportée au sprint 3 | US-013, X6 | — |
| [US-022](../US-022-ecran-admin-a-verifier.md) | Trancher les articles « à vérifier » | Back, Front, DevOps | Ferme | Livrée | US-019, US-021 | [#28](https://github.com/QVL-Collector/collector/pull/28) |
| [US-024](../US-024-controle-ferme-par-defaut.md) | Aucune annonce publiée ni perdue quand le contrôle est indisponible | Back, DevOps | Ferme | Livrée | US-013, US-015 | [#36](https://github.com/QVL-Collector/collector/pull/36) |
| [US-023](../US-023-catalogue-filtre-fiche.md) | Filtrer le catalogue et consulter la fiche d'un article | Back, Front | Ferme | Livrée | US-010, US-020 | [#24](https://github.com/QVL-Collector/collector/pull/24) |
| [US-025](../US-025-francais-anglais-accessibilite.md) | Utiliser Collector en anglais, avec des écrans accessibles | Front, DevOps | Ferme | Livrée | US-022, US-023 | [#20](https://github.com/QVL-Collector/collector/pull/20), [#33](https://github.com/QVL-Collector/collector/pull/33) |
| [US-026](../US-026-controle-echelle-file.md) | Absorber une vague de soumissions (KEDA sur le Contrôle) | DevOps, Test | Coupable | À faire, reportée au sprint 3 | US-016, US-024, X8 | — |
| [US-027](../US-027-limitation-debit.md) | Limiter le débit sur les routes sensibles | Back, DevOps | Coupable | Livrée | US-016 | [#35](https://github.com/QVL-Collector/collector/pull/35) |

## Capacité

| Élément | Valeur |
|---|---|
| Sessions de travail disponibles | Plan : 5 jours calendaires, 50 points ([BACKLOG.md](../BACKLOG.md) § 2) ; capacité non recalibrée à la clôture du sprint 1 |
| Engagé | 37 points fermes (11 US) et 5 points coupables (US-026, US-027) |
| Part réservée à la dette | 20 % (10 points) |
| Session AWS prévue (allumage / extinction) | 4 sessions prévues (≈ 5,4 $) ; aucune session du POC tenue, AWS bloqué jusqu'aux verdicts XP |
| **Réalisé** | 24 points livrés : 22 fermes (US-018, US-019, US-021, US-022, US-023, US-024, US-025) et 2 coupables (US-027) ; 18 points reportés (US-015, US-016, US-017, US-020, US-026) |

## Risques

Le fichier du sprint n'a pas été ouvert en planification : aucun registre de risques propre au sprint. Risque réalisé : le même que celui du sprint 1, verdicts XP en retard, qui bloque toute la partie « charge et autoscaling » (US-015 → US-016 → US-017, US-026) et la photo (US-020 : bucket S3 de la couche persistante, X6 validé par XP01).

Ordre de coupe si le sprint déborde : voir [D02](../../docs/decisions/D02-decoupage-applicatif.md) et [BACKLOG.md](../BACKLOG.md) § 5.

## Dette technique

| ID | Constat | Source (revue, audit, indicateur) | Impact | Remboursée (date, US) |
|---|---|---|---|---|
| DT-02 | Porte de couverture « pas de baisse » prévue par D12 dès le sprint 2 non active : étape « non bloquante » dans `ci.yml`, pas de `couverture-reference.json` | Revue d'US-030 (constat hors US) ; indicateur I4 | C6 : une baisse de couverture passe sans alerte | Non |
| DT-03 | Duplication (jscpd) et complexité front (`sonarjs`) prévues par D12 non outillées dans la CI | Indicateur I5 (relevé du sprint 2) | C6 : I5 réduit à la porte `gocognit` | Non |

## Revue (démo)

Aucune revue de sprint consignée à la date de clôture ; bilan établi le 2026-11-20 à partir des US, des rapports et de l'historique de `main`.

- **Parcours démontré** : mise en vente contrôlée complète **en local**, sans photo : coordonnées refusées (US-018), prix incohérents et vendeurs à risque mis « à vérifier », module Comptes minimal (US-019), revalidation de toute modification (US-021), écran admin avec décision sur une révision et 409 (US-022), catalogue filtré et fiche (US-023), contrôle fermé par défaut et reprise sans perte (US-024), français et anglais avec portes axe (US-025), limitation de débit (US-027).
- **Écarts constatés par rapport à l'objectif** :
  - aucune démonstration de charge ni d'autoscaling : I2 et I3 non mesurés, premier tir reporté au sprint 3 ;
  - pas de photo : US-020 non livrée ; US-023 est livrée alors que son AC2 exige des photos (« point à trancher » de la [revue d'US-023](../../docs/rapports/revues/US-023-revue-code.md), sans décision consignée) ;
  - contrôle manuel au clavier d'US-025 (AC5) « à tracer par Martin en fin de sprint » : grille non remplie ([rapport du lot 2](../../docs/rapports/tests/US-025-rapport-tests-lot-2.md)) ;
  - AC sans preuve sur environnement déployé : US-022 AC6 (second facteur admin) et parcours Playwright hors campagne, US-027 AC4 et AC5 (script curl contre l'ALB).
- **US non livrées et motif** : US-015, US-016, US-017, US-020 et US-026 (18 points). **AWS bloqué jusqu'aux verdicts XP** : US-015 dépend d'US-007, elle-même d'US-004 ; US-016, US-017 et US-026 dépendent d'US-015 ; US-020 demande le bucket de la couche persistante et X6. Déblocage par la **décision EKS du 2026-11-20** (D03, [PR #48](https://github.com/QVL-Collector/collector/pull/48)) ; reprise en sessions AWS 2 et 3 du sprint 3.
- **Ordre de coupe appliqué** ([BACKLOG.md](../BACKLOG.md) § 5) :
  - n° 3, US-027 reportée au sprint 3 : **non appliqué**, US-027 livrée (partie applicative ; preuves contre l'ALB en attente) ;
  - n° 4, US-026 coupée : **appliqué de fait**, par le blocage AWS et non par manque de capacité ;
  - n° 5, US-022 réduite : non appliqué, US-022 livrée sur ses AC hors environnement ;
  - n° 6, US-020 réduite à une photo : sans objet, US-020 entièrement reportée ;
  - le parcours soumission → contrôle → publication → consultation n'a pas été coupé.
- **Repli P1 du sprint 3** ([D02](../../docs/decisions/D02-decoupage-applicatif.md), complément du 2026-10-26 : « décidé à la revue du sprint 2 et consigné dans le bilan de sprint ») : **non déclenché**. Constat : Interactions et Transactions ont été développés au sprint 3 (US-028 à US-032 livrées). Aucune décision explicite de Martin n'est consignée à cette revue.

### Décisions prises pendant le sprint

| Date | Décision | Où elle est consignée |
|---|---|---|
| 2026-11-06 | Filet de l'outbox : délai doublé à chaque relance, plafonné à 1 h, sans arrêt (arbitrage demandé par la revue d'US-024) | [D05](../../docs/decisions/D05-communication-interservice.md), « Précisé le 2026-11-06 (US-024) » |
| 2026-11-06 | Rejeu : sous-commande `rejouer --depuis` obligatoire, fenêtre de 7 jours au plus (MC9) | [D05](../../docs/decisions/D05-communication-interservice.md), MC9 |
| 2026-11-06 | Idempotence d'Interactions sur l'`id` de l'enveloppe, conservé au rejeu (préparation d'US-028) | [D05](../../docs/decisions/D05-communication-interservice.md), « Précisé le 2026-11-06 (US-028) » |
| Non datée | Suspension d'un compte par sous-commande, module Comptes interne sans API (US-019 lot 2) | [Revue d'US-019 lot 2](../../docs/rapports/revues/US-019-revue-code-lot-2.md) seulement |

Expérimentation pendant le sprint : session EKS 1 de XP01 ([PR #34](https://github.com/QVL-Collector/collector/pull/34), 2026-11-06), sans verdict.

## Rétrospective

Aucune rétrospective consignée. Constats factuels proposés, à valider par Martin :

| À garder | À améliorer | Action pour le sprint suivant |
|---|---|---|
| Tests sur PostgreSQL réel (testcontainers) et parcours Playwright avec axe en français et en anglais | US coupable (US-027) livrée avant des US fermes bloquées : l'ordre de coupe ne couvre pas un blocage d'environnement | À trancher par Martin |
| Arbitrages de revue reportés dans la décision concernée (D05) | Décisions de Martin restées dans les rapports (US-019, US-023) | À trancher par Martin |

## Indicateurs relevés

Relevé détaillé : [indicateurs-sprint-2.md](../../docs/mesures/indicateurs-sprint-2.md).

| Indicateur | Valeur | Tendance | Axe d'amélioration |
|---|---|---|---|
| I1 — AC couverts par un test vert | Non mesuré (relevé non généré) ; AC sans preuve nommés dans le relevé | Non calculable | Preuves après déploiement d'US-022, US-027 |
| I2 — Latence p95 et mise à l'échelle | Non mesuré : aucun tir (US-016 reportée) | Non calculable | Session AWS 3 |
| I3 — Fiabilité sous perte d'instance | Non mesuré : aucun tir (US-017 reportée) | Non calculable | Session AWS 3 |
| I4 — Couverture par service | Valeurs ponctuelles des rapports (ex. Contrôle `regles` 100 %, `coordonnees` 98,5 %, `canal` 61,4 %) ; porte « pas de baisse » absente (DT-02) | Non calculable | `services/controle/internal/canal` |
| I5 — Complexité cognitive et duplication | Non mesuré (porte `gocognit` seule) | Non calculable | DT-03 |
| I6 — Vulnérabilités et exceptions | 0 exception dans `.trivyignore` ; vulnérabilités ouvertes non relevées | Stable (exceptions) | Lire l'artefact `i6` de `main` |
| I7 — Livraison (DORA réduit) | Non mesuré ; 10 PR d'US mergées | Non calculable | DT-01 |
| Vélocité (points livrés) | 24 points (22 fermes, 2 coupables) | Non comparable (blocage AWS) | Recalibrage : voir « Clôture » |

## Clôture

- [ ] Audits : [architecture](../../docs/audits/sprint-2-architecture.md), [sécurité](../../docs/audits/sprint-2-securite.md) — non faits ; regroupés en J6 (décision de pilotage)
- [ ] Documentation touchée à jour — D05 précisée ; `docs/exploitation.md` inexistant
- [ ] US livrées mergées sur `main`, branches mergées supprimées — US livrées mergées ; branches distantes `feat/US-019-*`, `feat/US-021-*`, `feat/US-022-*`, `feat/US-023-*`, `feat/US-024-*` et branche locale `feat/US-018-filtrage-coordonnees` encore présentes
- [x] Infrastructure éteinte, aucune ressource restante — aucune ressource du POC allumée pendant le sprint (extinction de la session EKS 1 de XP01 suivie dans [XP01](../../docs/experimentations/XP01-orchestrateur.md), critère G3)
