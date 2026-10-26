# Schéma 04 — Flux de mise en vente : soumission, contrôle, décision, diffusion

Document parent : [architecture-technique.md](../architecture-technique.md) § 4.1. Décisions : [D01](../../decisions/D01-fonctionnalite-poc.md), [D02](../../decisions/D02-decoupage-applicatif.md), [D05](../../decisions/D05-communication-interservice.md), [D06](../../decisions/D06-authentification-autorisation.md), [D07](../../decisions/D07-donnees.md).
Le passage par l'ALB n'est pas représenté. Les messages passent par RabbitMQ (statut **Candidate**, XP02).

```mermaid
sequenceDiagram
  autonumber
  actor VD as Vendeur
  participant KC as Keycloak
  participant PL as Place de marché
  participant DB as Base marche
  participant S3 as S3 photos
  participant MQ as RabbitMQ
  participant CT as Contrôle
  participant IN as Interactions
  actor AD as Admin
  actor VI as Visiteur

  VD->>KC: Connexion OIDC avec PKCE
  KC-->>VD: Jeton d'accès 5 min, rôle vendeur, en mémoire
  VD->>PL: POST article, Authorization Bearer
  Note right of PL: jwx : signature, iss, aud, exp, azp collector-web, rôle, puis propriété de la boutique
  PL->>DB: Article « brouillon »
  VD->>PL: Déposer une photo, corps borné
  PL->>PL: JPEG ou PNG seulement, ré-encodage sans métadonnées, nom aléatoire
  PL->>S3: Objet privé, par le rôle du nœud
  VD->>PL: Soumettre l'article
  PL->>DB: Une transaction : état « en contrôle », révision n, ligne d'outbox
  PL-->>VD: 202, « en contrôle »
  PL-)MQ: Relais d'outbox : demande de contrôle v1 sur controle.demandes
  MQ-)CT: Livrer, après validation taille, clés, schéma
  CT->>CT: Règles : complétude, coordonnées, plage de prix, statut vendeur
  CT-)MQ: Résultat v1 sur controle.resultats, révision n, issue, motif, version des règles
  CT->>MQ: Acquitter la demande après confirmation du broker
  MQ-)PL: Livrer sur pdm.resultats-controle
  PL->>DB: Appliquer si « en contrôle » à la révision n, sinon ignorer
  alt Publié
    PL->>DB: État « publié » et ligne d'outbox de la décision
  else Refusé
    PL->>DB: État « refusé », motif catégorisé, ligne d'outbox
  else À vérifier
    AD->>PL: Décision pour la révision examinée, OTP exigé à la connexion
    alt Révision examinée courante
      PL->>DB: Décision journalisée en audit et ligne d'outbox
    else Article modifié entre-temps
      PL-->>AD: 409, recharger l'article
    end
  end
  PL-)MQ: Relais d'outbox : décision v1 sur pdm.evenements, clé publication.decidee.v1
  MQ-)IN: Livrer sur interactions.publication-decidee
  IN->>IN: Notification créée, contrainte d'unicité par décision
  VD->>IN: Lister ses notifications
  VI->>PL: Catalogue, puis fiche article, sans connexion
  PL-->>VI: Articles publiés seuls, URL présignées de 15 min au plus
  VI->>S3: Photos par URL présignée, origine distincte, sans cookie
```

Points de lecture :
- **Aucune perte** : l'événement est écrit dans la même transaction que le changement d'état (outbox) ; le relais ne marque une ligne publiée qu'après la confirmation du broker. Un article resté « en contrôle » au-delà d'un délai est renvoyé au contrôle (filet de la file, D05).
- **Aucun doublon d'effet** : la Place de marché n'applique un résultat qu'à la révision courante ; Interactions refuse une seconde notification pour la même décision.
- **Message non conforme** : écarté sans nouvel essai, vers la file `.ecartes` de sa file, avec une alerte (ES-31, D09).
- **Émetteur vérifié** : chaque producteur pose `user-id`, vérifié par RabbitMQ contre l'identité connectée, puis par le consommateur (MC2).
- Revalidation après modification et contrôle indisponible : [architecture-technique.md](../architecture-technique.md) § 4.2.
