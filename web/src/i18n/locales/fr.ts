export const fr = {
  app: {
    name: 'Collector',
    homeLink: 'Collector, retour au catalogue',
    skipToContent: 'Aller au contenu principal',
    language: {
      label: 'Langue de l’interface',
      frShort: 'FR',
      frFull: 'Français',
      enShort: 'EN',
      enFull: 'English',
    },
    bootstrapFailure: {
      title: 'L’application n’a pas pu démarrer',
      description: 'Rechargez la page dans quelques instants.',
    },
  },
  auth: {
    signIn: 'Se connecter',
    signOut: 'Se déconnecter',
    accountLabel: 'Votre compte',
    unknownUser: 'Utilisateur connecté',
    signInFailed: 'Le service de connexion est injoignable. Réessayez dans quelques instants.',
    signOutFailed:
      'La déconnexion n’a pas pu être confirmée par le service de connexion. Ce navigateur a oublié votre session.',
    roles: {
      admin: 'Administrateur',
      vendeur: 'Vendeur',
      acheteur: 'Acheteur',
      none: 'Sans rôle',
    },
    callback: {
      documentTitle: 'Connexion · Collector',
      pending: 'Connexion en cours…',
      title: 'La connexion n’a pas abouti',
      description:
        'Votre session n’a pas pu être ouverte. Reprenez la connexion avec le bouton « Se connecter ».',
    },
    gate: {
      redirecting: 'Redirection vers la page de connexion…',
      unavailableTitle: 'Le service de connexion est indisponible',
      unavailableDescription: 'Vérifiez votre connexion, puis réessayez.',
      retry: 'Réessayer',
    },
  },
  accessDenied: {
    documentTitle: 'Accès réservé · Collector',
    title: 'Accès réservé',
    description: 'Votre compte n’a pas le rôle requis pour ouvrir cet espace.',
  },
  seller: {
    documentTitle: 'Espace vendeur · Collector',
    title: 'Espace vendeur',
    welcome: 'Bienvenue, {{name}}. Vos annonces apparaîtront ici.',
    navigation: 'Espace vendeur',
    untitled: 'Article sans titre',
    notProvided: 'Non renseigné',
    newArticle: 'Nouvel article',
    backToList: 'Retour à mes articles',
    states: {
      brouillon: 'Brouillon',
      en_controle: 'En contrôle',
      publie: 'Publié',
      refuse: 'Refusé',
      a_verifier: 'À vérifier',
    },
    stateDetails: {
      brouillon: 'Invisible au catalogue tant que l’annonce n’est pas soumise puis publiée.',
      en_controle:
        'Contrôle automatique en cours. Cette page se met à jour d’elle-même dès le résultat.',
      publie: 'L’annonce est visible dans le catalogue.',
      refuse: 'Corrigez l’annonce, enregistrez-la, puis soumettez-la de nouveau.',
      a_verifier:
        'Un administrateur doit examiner l’annonce. Elle reste invisible au catalogue d’ici là.',
    },
    reason: 'Motif : {{reason}}',
    announcements: {
      stateChanged: '« {{title}} » : {{state}}.',
      stateChangedWithReason: '« {{title}} » : {{state}}, motif : {{reason}}.',
    },
    list: {
      title: 'Mes articles',
      paginationLabel: 'Pagination de mes articles',
      loading: 'Chargement de vos articles en cours',
      loaded_one: '{{count}} article',
      loaded_other: '{{count}} articles',
      updatedAt: 'Modifié le {{date}}',
      noPrice: 'Prix non renseigné',
      empty: {
        title: 'Aucun article pour le moment',
        description:
          'Créez un brouillon : il reste invisible au catalogue jusqu’à sa publication.',
      },
      error: {
        title: 'Vos articles sont momentanément indisponibles',
        description: 'Vérifiez votre connexion, puis réessayez.',
        retry: 'Réessayer',
      },
    },
    editor: {
      newDocumentTitle: 'Nouvel article · Collector',
      newSubtitle:
        'Préparez votre annonce. Elle reste un brouillon invisible au catalogue tant que vous ne la soumettez pas.',
      documentTitle: '{{title}} · Collector',
      loading: 'Chargement de l’article en cours',
      readOnly: 'L’annonce n’est pas modifiable dans son état actuel.',
      notFound: {
        title: 'Article introuvable',
        description: 'Cet article n’existe pas ou n’appartient pas à votre compte.',
      },
      error: {
        title: 'L’article est momentanément indisponible',
        description: 'Vérifiez votre connexion, puis réessayez.',
        retry: 'Réessayer',
      },
    },
    form: {
      label: 'Contenu de l’annonce',
      hint:
        'Un brouillon peut rester incomplet. Titre, description, catégorie et prix sont exigés pour la publication.',
      title: 'Titre',
      description: 'Description',
      category: 'Catégorie',
      noCategory: 'Aucune catégorie',
      categoriesUnavailable: 'Les catégories sont momentanément indisponibles.',
      price: 'Prix (€)',
      shippingCost: 'Frais de port (€)',
      amountHint: 'Exemple : 12,50',
      characterCount: '{{length}} sur {{max}} caractères',
      create: 'Créer le brouillon',
      save: 'Enregistrer le brouillon',
      saved: 'Brouillon enregistré, révision {{revision}}.',
      invalid_one: 'Le formulaire contient {{count}} erreur : corrigez le champ signalé.',
      invalid_other: 'Le formulaire contient {{count}} erreurs : corrigez les champs signalés.',
      locked:
        'L’annonce n’est plus modifiable dans son état actuel. La page a été mise à jour.',
      failure: 'L’enregistrement a échoué. Vérifiez votre connexion, puis réessayez.',
      conflict: {
        title: 'Cet article a été modifié entre-temps',
        description:
          'Il a changé depuis son ouverture (révision {{revision}}), par exemple dans un autre onglet. Vos modifications n’ont pas été enregistrées pour ne rien écraser : rechargez la dernière version, puis refaites vos changements.',
        reload: 'Recharger la dernière version',
      },
      errors: {
        obligatoire: 'Ce champ est obligatoire.',
        invalide: 'Valeur invalide. Exemple de montant : 12,50.',
        hors_limites: 'Valeur hors des limites autorisées.',
        trop_long: 'Texte trop long.',
        caractere_interdit: 'Ce texte contient un caractère non autorisé.',
        negatif: 'Le montant ne peut pas être négatif.',
        au_dela_du_plafond: 'Montant au-delà du plafond autorisé.',
        devise_non_prise_en_charge: 'Devise non prise en charge.',
        categorie_inconnue: 'Catégorie inconnue.',
      },
    },
    submission: {
      title: 'État de l’annonce',
      submit: 'Soumettre au contrôle',
      unsavedChanges: 'Enregistrez vos modifications avant de soumettre l’annonce.',
      polling: 'Contrôle en cours, mise à jour automatique…',
      conflict:
        'L’article a changé depuis son affichage. Il vient d’être rechargé : vérifiez-le, puis soumettez-le de nouveau.',
      locked:
        'L’annonce ne peut pas être soumise dans son état actuel. Elle vient d’être rechargée.',
      failure: 'La soumission a échoué. Vérifiez votre connexion, puis réessayez.',
    },
  },
  common: {
    backToCatalogue: 'Retour au catalogue',
    pagination: {
      page: 'Aller à la page {{page}}',
      currentPage: 'Page {{page}}, page actuelle',
      first: 'Première page',
      last: 'Dernière page',
      previous: 'Page précédente',
      next: 'Page suivante',
    },
  },
  catalogue: {
    documentTitle: 'Catalogue · Collector',
    title: 'Catalogue',
    subtitle:
      'Des objets de collection publiés par des particuliers, vérifiés avant leur mise en ligne.',
    listLabel: 'Articles publiés',
    paginationLabel: 'Pagination du catalogue',
    loading: 'Chargement du catalogue en cours',
    loaded_one: '{{count}} article publié, page {{page}} sur {{pageCount}}',
    loaded_other: '{{count}} articles publiés, page {{page}} sur {{pageCount}}',
    empty: {
      title: 'Aucun article publié pour le moment',
      description: 'De nouveaux objets arrivent régulièrement : revenez bientôt.',
    },
    outOfRange: {
      title: 'Cette page du catalogue est vide',
      description: 'Le catalogue compte moins de pages que celle demandée.',
      action: 'Revenir à la première page',
    },
    error: {
      title: 'Le catalogue est momentanément indisponible',
      description: 'Vérifiez votre connexion, puis réessayez.',
      retry: 'Réessayer',
    },
  },
  control: {
    reasons: {
      annonce_incomplete: 'Annonce incomplète',
      coordonnees_detectees: 'Coordonnées détectées',
      compte_non_autorise: 'Compte non autorisé à vendre',
      prix_a_confirmer: 'Prix à confirmer',
      vendeur_a_verifier: 'Vendeur à vérifier',
      categorie_inconnue: 'Catégorie inconnue',
    },
  },
  notFound: {
    documentTitle: 'Page introuvable · Collector',
    title: 'Page introuvable',
    description: 'L’adresse demandée ne correspond à aucune page.',
  },
  routeError: {
    documentTitle: 'Erreur · Collector',
    title: 'Un problème est survenu',
    description: 'Cette page n’a pas pu s’afficher.',
  },
};

type TranslationTree<T> = {
  readonly [Key in keyof T]: T[Key] extends string ? string : TranslationTree<T[Key]>;
};

export type Translation = TranslationTree<typeof fr>;
