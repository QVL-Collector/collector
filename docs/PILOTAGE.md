# Pilotage du projet Collector

Ce document fixe les règles de travail du projet : qui fait quoi, comment une décision est prise, comment une expérimentation est menée, comment une user story (US) est développée puis livrée.
En cas de doute, ce document fait foi. Toute évolution de ces règles est validée par le Lead Developer.

## 1. Cadrage

- **Nature** : projet d'évaluation CESI, bloc INFMAALSIA2V (« Superviser et assurer le développement des applications logicielles »). Livrable : un POC de Collector.shop.
- **Évaluation** : seule la grille individuelle compte. Elle comporte 10 critères (C1 à C10), objectif 5/5 sur chacun. Chaque décision et chaque US indique les critères qu'elle sert.
- **Décideur** : Martin, Lead Developer. Il tranche toutes les décisions et valide toutes les livraisons.
- **Hébergement** : AWS Academy Learner Lab, crédits inférieurs à 100 $. L'infrastructure est allumée puis éteinte via IaC et scripts (`tools/`) : rien ne tourne en continu.
- **Contraintes AWS probables, à vérifier** avant la première expérimentation cloud : rôle IAM imposé (LabRole, pas de création de rôles IAM ni de fournisseur OIDC), régions limitées, sessions à durée limitée avec credentials temporaires, budget plafonné.
- **Dépôt** : monorepo. La carte des documents est dans le [README](../README.md).
- **Consigne** : les documents sources restent hors du dépôt et ne sont jamais modifiés. Leur synthèse est dans `docs/00-cadrage/`.
- **Simplicité d'abord** : à besoin égal, la solution la plus simple l'emporte. Toute complexité ajoutée est justifiée par une exigence ou un critère de la grille.

### Jalons

| Jalon | Contenu |
|---|---|
| J0 | Organisation |
| J1 | Besoin et découpage (D01, D02) |
| J2 | Décisions techniques (D03 à D14) |
| J3 | Documents de process, bac à sable, configuration AWS |
| J4 | Sprint 1 : walking skeleton (WS-2, § 3.4) : Place de marché et Contrôle de publication à une règle, reliés par le canal |
| J5 | Sprint 2 : mise en vente contrôlée complète, charge et autoscaling |
| J5 bis | Sprint 3 : diffusion vers Interactions et réservation par Transactions |
| J6 | Audit sécurité, remédiation, analyse de la dette, compétences |
| J7 | Répétitions, vidéo de secours, soutenance |

Contenu indicatif des trois sprints et ordre de coupe : [D02](decisions/D02-decoupage-applicatif.md), « Découpage indicatif en sprints ».
Le sprint 3 est inséré avant J6 sous le numéro J5 bis : les jalons J6 et J7, cités dans d'autres documents, gardent leur numéro.

## 2. Rôles

Les rôles sont des fonctions. Une même personne peut en tenir plusieurs, mais chaque étape d'un circuit est rattachée à une fonction précise.

| Fonction | Responsabilités | Ne fait pas |
|---|---|---|
| Lead Developer / décideur (Martin) | Tranche les décisions, exécute les expérimentations, teste et valide les US, prononce la livraison | — |
| Architecture | Rédige les options des décisions d'architecture, donne l'avis simplicité, relit les expérimentations, audit d'architecture en fin de sprint | Ne développe pas |
| Sécurité | Avis sécurité sur chaque décision, relecture IAM et TLS des expérimentations, audit sécurité en fin de sprint | Ne développe pas |
| Développement back | Implémente les US côté services, avis faisabilité sur les décisions qui le concernent | Ne tranche pas les décisions |
| Développement front | Implémente les US côté web, avis faisabilité sur les décisions qui le concernent | Ne tranche pas les décisions |
| Test | Écrit et exécute les tests à partir des AC du fichier US, jamais à partir du code | Ne corrige pas le code |
| Revue de code | Relit le diff de la branche en fin d'US, renvoie les corrections vers le développement | Ne corrige pas le code |
| DevOps | Dépôt, branches, merge, code d'infra (IaC, Dockerfiles, manifests de déploiement, pipeline CI/CD, scripts `sandbox/` et `tools/`), protocoles d'expérimentation, US d'infra, avis faisabilité sur les décisions de plateforme | N'écrit jamais de code applicatif |

## 3. Circuits

### 3.1 Brainstorm — une décision

1. La fonction concernée (le plus souvent architecture) crée `docs/decisions/Dxx-slug.md` à partir du [gabarit](decisions/_GABARIT.md) : contexte, options, recommandation. Statut `Proposée`.
2. Avis croisés, collés dans les sections Avis du fichier : sécurité (section « Avis sécurité ») ; faisabilité et simplicité (section « Avis faisabilité »).
3. Martin tranche. Sa décision est consignée avec ses propres mots dans la section « Décision de Martin ».
4. Le statut est mis à jour (`Décidée`, `Rejetée` ou `Candidate (XPnn)`), ainsi que l'[index](decisions/INDEX.md).
5. Commit : `docs(decisions): record dxx <slug>` (sujet en minuscules, règle Karma).

Règles :
- Une décision n'est jamais écrite comme prise tant que Martin ne l'a pas tranchée.
- Les décisions sont instruites par session, en respectant la colonne « Dépend de » de l'index : S1 = D01 · S2 = D02 · S3 = D03 + D04 · S4 = D05 à D11 · S5 = D12 à D14.
- Une décision de plateforme qui demande une vérification sur le terrain est prise en statut `Candidate (XPnn)`, puis confirmée ou rejetée après l'expérimentation.

### 3.2 Expérimentation (XP)

1. DevOps rédige le protocole `docs/experimentations/XPnn-slug.md` à partir du [gabarit](experimentations/_GABARIT.md) et écrit les scripts dans `sandbox/XPnn-slug/`. Les critères de succès mesurables et les versions exactes sont fixés **avant** l'exécution.
2. Martin exécute lui-même l'expérimentation et note les difficultés et les limites.
3. Les résultats sont complétés dans le protocole.
4. Relecture architecture (pertinence et conclusions) et sécurité (IAM, TLS).
5. Martin rend le verdict : la décision liée passe de `Candidate (XPnn)` à `Décidée` ou `Rejetée`. Le fichier de décision et l'index sont mis à jour.

Règles :
- Expérimentations locales d'abord.
- Une session de configuration AWS avec Martin a lieu avant la première expérimentation cloud.
- Après chaque expérimentation cloud : extinction (`tools/down.sh`) et vérification qu'aucune ressource ne reste allumée.
- Le bac à sable est jetable : aucun code du POC n'y vit, et le POC ne dépend jamais du bac à sable.

### 3.3 Dev — une US

1. **Développement** (back, front, ou DevOps pour une US d'infra) sur la branche `feat/US-xxx-slug`. Statut `En cours`.
2. **Test** sur les AC du fichier US. Statut `En test`.
3. **Revue de code** du diff de la branche. Statut `En review`.
   Tout retour de test ou de revue renvoie l'US à l'étape 1, jusqu'à ce que test et revue soient verts.
4. **Test par Martin.**
5. Sur la phrase explicite de Martin « je valide la livraison de la feature » : statut `Validée Martin`, PR vers `main`, CI verte, merge. Statut `Livrée`, date et liens renseignés dans la section Suivi de l'US.

### 3.4 Fin de sprint

1. Audits : `docs/audits/sprint-N-architecture.md` et `docs/audits/sprint-N-securite.md`.
2. Relevé des indicateurs : `docs/mesures/indicateurs-sprint-N.md`.
3. Mise à jour de la documentation touchée par le sprint.
4. Clôture : toutes les US livrées sont mergées sur `main`, les branches mergées sont supprimées, les statuts sont à jour. Une branche non mergée est signalée, jamais supprimée d'office.

Le **sprint 1 est un walking skeleton**, variante WS-2 décidée dans [D02](decisions/D02-decoupage-applicatif.md) :
- **Contenu** : la Place de marché et le Contrôle de publication, reliés par le canal, déployés de bout en bout sur l'environnement managé, avec CI, HTTPS, authentification, allumage et extinction depuis zéro.
- **Une seule règle de contrôle** : la complétude du titre, de la description, de la catégorie et du prix.
- Le squelette porte la première tranche de la fonctionnalité métier, réutilisée telle quelle au sprint 2, qui ajoute les autres règles et les écrans.
- **Conditions** : l'expérimentation du canal est concluante sur l'environnement managé, et D05 est tranchée avant J4.
- **À défaut**, variante WS-1 : la Place de marché seule ; le canal et le Contrôle arrivent au sprint 2. Aucune publication directe temporaire : un article n'est jamais publié sans contrôle.

Le contenu et les objectifs de chaque sprint sont dans `backlog/sprints/sprint-N.md`.

## 4. Statuts

### Décision

| Statut | Signification |
|---|---|
| À instruire | Listée dans l'index, pas encore de fichier |
| Proposée | Options et recommandation rédigées, en attente des avis puis de Martin |
| Candidate (XPnn) | Option retenue provisoirement, à confirmer par l'expérimentation XPnn |
| Décidée | Tranchée par Martin |
| Rejetée | Écartée par Martin, directement ou après expérimentation |
| Remplacée (par Dyy) | Remplacée par une décision ultérieure ; le fichier est conservé |

### US

| Statut | Signification |
|---|---|
| À faire | Rédigée, pas commencée |
| En cours | En développement sur sa branche |
| En test | Tests sur les AC en cours |
| En review | Revue de code en cours |
| Validée Martin | Martin a prononcé « je valide la livraison de la feature » |
| Livrée | Mergée sur `main` avec CI verte |

## 5. Definition of Done d'une US

Une US est terminée quand **toutes** les conditions suivantes sont vraies :

- [ ] Tous les AC (fonctionnels et non fonctionnels) sont couverts par des tests verts en CI.
- [ ] La revue de code est faite et ses retours sont traités.
- [ ] Martin a testé et validé la livraison (« je valide la livraison de la feature »).
- [ ] La documentation touchée est à jour (README, `docs/exploitation.md`, documents concernés).
- [ ] Le statut de l'US est mis à jour dans son fichier, avec la date de validation et les liens vers les rapports de test et de revue.
- [ ] Aucune donnée sensible ni credential n'a été ajouté au dépôt.

## 6. Conventions git

- **Branche principale** : `main`. Elle ne reçoit que du travail validé ; son historique n'est jamais réécrit.
- **Une branche par US** : `feat/US-xxx-slug`, créée depuis `main`. Pas de développement directement sur `main`.
- **Commits au format Karma** : `<type>(<scope>): <sujet>`.
  - Types : `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `build`, `ci`, `revert`.
  - Sujet à l'impératif, en minuscules, sans point final. Corps optionnel séparé par une ligne vide.
  - Exemples : `chore(repo): initialize collector project structure`, `docs(decisions): record d01 poc business feature`.
- **Messages de commit et de PR** : ils décrivent le changement, sans ligne de signature, de co-auteur ou de mention d'outil.
- **Merge sur `main`** : uniquement après la phrase explicite de Martin « je valide la livraison de la feature ». Une formulation ambiguë ne vaut pas validation : on redemande.
- **PR** : vers `main`, CI verte obligatoire avant merge. La forge et le pipeline sont définis par D10.
- **Après merge** : la branche de l'US est supprimée.

## 7. Données sensibles

- **Aucune donnée sensible ni credential dans le dépôt** : clés et sessions AWS (y compris les credentials temporaires du Learner Lab), mots de passe, tokens, clés privées, certificats, fichiers `.env`, states IaC.
- Les variables d'environnement sont documentées **par leur nom seulement** dans `docs/exploitation.md`. Un fichier `.env.example` peut exister, avec des valeurs fictives.
- Les jeux de données de test sont fictifs.
- Le `.gitignore` bloque les cas courants mais ne remplace pas la vigilance : relire `git status` et le diff avant chaque commit.
- En cas de fuite : prévenir Martin immédiatement, révoquer ou renouveler le secret, puis nettoyer l'historique.

## 8. Formats des documents

| Document | Gabarit | Emplacement et nommage |
|---|---|---|
| Décision | [decisions/_GABARIT.md](decisions/_GABARIT.md) | `docs/decisions/Dxx-slug.md`, référencée dans [INDEX.md](decisions/INDEX.md) |
| Expérimentation | [experimentations/_GABARIT.md](experimentations/_GABARIT.md) | `docs/experimentations/XPnn-slug.md` + scripts dans `sandbox/XPnn-slug/` |
| User story | [../backlog/_GABARIT-US.md](../backlog/_GABARIT-US.md) | `backlog/US-xxx-slug.md` |
| Sprint | — | `backlog/sprints/sprint-N.md` |
| Audit de fin de sprint | — | `docs/audits/sprint-N-architecture.md`, `docs/audits/sprint-N-securite.md` |
| Relevé d'indicateurs | — | `docs/mesures/indicateurs-sprint-N.md` |

Règles de rédaction :
- Court et structuré : titres, listes, tableaux. Une phrase = une idée.
- Une information vit à un seul endroit ; ailleurs, on met un lien.
- Dates au format `AAAA-MM-JJ`.
- Le contenu obsolète est supprimé ou condensé, pas empilé. Une décision remplacée garde son fichier avec le statut `Remplacée (par Dyy)`.
