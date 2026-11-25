export const fr = {
  app: {
    name: 'Collector',
    homeLink: 'Collector, retour au catalogue',
    skipToContent: 'Aller au contenu principal',
    language: {
      label: 'Langue de l’interface',
      frShort: 'FR',
      frFull: 'FR, Français',
      enShort: 'EN',
      enFull: 'EN, English',
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
        description: 'Créez un brouillon : il reste invisible au catalogue jusqu’à sa publication.',
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
      hint: 'Un brouillon peut rester incomplet. Titre, description, catégorie et prix sont exigés pour la publication.',
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
      saveForControl: 'Enregistrer et renvoyer au contrôle',
      savedForControl:
        'Modifications enregistrées, révision {{revision}} : l’annonce repasse au contrôle.',
      recontrol: {
        published: {
          title: 'L’annonce sera retirée du catalogue',
          description:
            'Toute modification enregistrée, même du seul titre, retire l’annonce du catalogue le temps du contrôle. Elle y revient une fois la nouvelle version acceptée.',
        },
        underReview: {
          title: 'Le contrôle repartira sur la nouvelle version',
          description:
            'Toute modification enregistrée relance le contrôle sur votre nouvelle version : le résultat attendu pour la version actuelle sera ignoré. L’annonce reste invisible au catalogue d’ici là.',
        },
      },
      invalid_one: 'Le formulaire contient {{count}} erreur : corrigez le champ signalé.',
      invalid_other: 'Le formulaire contient {{count}} erreurs : corrigez les champs signalés.',
      locked: 'L’annonce n’est plus modifiable dans son état actuel. La page a été mise à jour.',
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
    reserved:
      'Cet article est réservé par un acheteur ou déjà vendu : il ne peut être ni modifié ni soumis pendant la réservation, ni après la vente.',
  },
  notifications: {
    documentTitle: 'Notifications · Collector',
    navigation: 'Notifications',
    title: 'Notifications',
    subtitle:
      'Chaque décision prise sur vos articles apparaît ici : publication, refus ou examen par un administrateur.',
    listTitle: 'Décisions sur vos articles',
    loading: 'Chargement de vos notifications en cours',
    loaded_one: '{{count}} notification',
    loaded_other: '{{count}} notifications',
    decidedAt: 'Décision du {{date}}',
    revision: 'Révision {{revision}}',
    issues: {
      publie: {
        title: 'Votre article a été publié',
        description: 'Il est désormais visible dans le catalogue.',
      },
      refuse: {
        title: 'Votre article a été refusé',
        description: 'Corrigez l’annonce, enregistrez-la, puis soumettez-la de nouveau.',
      },
      a_verifier: {
        title: 'Votre article doit être vérifié',
        description: 'Un administrateur va l’examiner. Il reste invisible au catalogue d’ici là.',
      },
    },
    empty: {
      title: 'Aucune notification pour le moment',
      description:
        'Vous serez prévenu ici dès qu’une décision sera prise sur l’un de vos articles.',
    },
    error: {
      title: 'Vos notifications sont momentanément indisponibles',
      description: 'Vérifiez votre connexion, puis réessayez.',
      retry: 'Réessayer',
    },
    unauthenticated: {
      title: 'Votre session a expiré',
      description: 'Reconnectez-vous pour consulter vos notifications.',
      signIn: 'Se reconnecter',
    },
    forbidden: {
      title: 'Accès réservé',
      description: 'Votre compte n’a pas le rôle requis pour consulter ces notifications.',
    },
  },
  orders: {
    navigation: 'Mes réservations',
    list: {
      documentTitle: 'Mes réservations · Collector',
      title: 'Mes réservations',
      subtitle:
        'Retrouvez les articles réservés à votre nom, leur montant et le temps qu’il vous reste pour payer.',
      listTitle: 'Vos réservations',
      listLabel: 'Réservations',
      paginationLabel: 'Pagination de mes réservations',
      loading: 'Chargement de vos réservations en cours',
      loaded_one: '{{count}} réservation',
      loaded_other: '{{count}} réservations',
      total: 'Montant total',
      timeLeft: 'Expire {{relative}}',
      lastMinute: 'Expire dans moins d’une minute',
      deadlineReached: 'Échéance atteinte, mise à jour de l’état en cours…',
      expired: 'La réservation de « {{title}} » a expiré.',
      viewDetail: 'Voir la réservation',
      viewDetailLabel: 'Voir la réservation « {{title}} »',
      forbiddenDescription: 'Votre compte n’a pas le rôle requis pour consulter ces réservations.',
      empty: {
        title: 'Aucune réservation pour le moment',
        description:
          'Réservez un article depuis le catalogue : il vous est garanti le temps du paiement.',
        action: 'Parcourir le catalogue',
      },
      error: {
        title: 'Vos réservations sont momentanément indisponibles',
        description: 'Vérifiez votre connexion, puis réessayez.',
        retry: 'Réessayer',
      },
      unauthenticated: {
        description: 'Reconnectez-vous pour consulter vos réservations.',
      },
    },
    pendingDocumentTitle: 'Votre commande · Collector',
    pendingTitle: 'Votre commande',
    loading: 'Chargement de votre commande en cours',
    views: {
      en_attente_paiement: {
        documentTitle: 'Article réservé · Collector',
        title: 'Article réservé',
        subtitle:
          'Votre commande est enregistrée. L’article vous est garanti jusqu’à l’échéance de paiement.',
        announcement: 'Réservation confirmée : commande en attente de paiement',
      },
      expiree: {
        documentTitle: 'Réservation expirée · Collector',
        title: 'Réservation expirée',
        subtitle: 'La réservation a expiré, l’article est de nouveau disponible.',
        announcement:
          'Commande expirée : la réservation a expiré, l’article est de nouveau disponible',
      },
    },
    statusTitle: 'Statut de la commande',
    statuses: {
      en_attente_paiement: 'En attente de paiement',
      expiree: 'Expirée',
    },
    createdAt: 'Commande passée le {{date}}',
    deadlines: {
      en_attente_paiement: {
        title: 'À régler avant le',
        hint: 'Échéance fixée par la place de marché.',
      },
      expiree: {
        title: 'Échéance dépassée le',
        hint: 'L’article n’est plus réservé à votre nom : réservez-le de nouveau depuis sa fiche s’il est toujours disponible.',
      },
    },
    amountsTitle: 'Montants',
    price: 'Prix de référence',
    shippingCost: 'Frais de port',
    totalPaid: 'Total payé',
    splitTitle: 'Répartition du total payé',
    commission: 'Commission Collector (5 % du total)',
    sellerAmount: 'Montant dû au vendeur',
    commissionHint:
      'La commission est prélevée sur le total payé : elle ne s’ajoute pas à votre paiement. Le vendeur reçoit le reste.',
    viewArticle: 'Voir l’article',
    notFound: {
      documentTitle: 'Commande introuvable · Collector',
      title: 'Commande introuvable',
      description: 'Cette commande n’existe pas ou n’est pas liée à votre compte.',
    },
    error: {
      title: 'Votre commande est momentanément indisponible',
      description: 'Vérifiez votre connexion, puis réessayez.',
      retry: 'Réessayer',
    },
    unauthenticated: {
      title: 'Votre session a expiré',
      description: 'Reconnectez-vous pour consulter votre commande.',
      signIn: 'Se reconnecter',
    },
    forbidden: {
      title: 'Accès réservé',
      description: 'Votre compte n’a pas le rôle requis pour consulter cette commande.',
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
    loadedInCategory_one:
      '{{count}} article publié dans la catégorie {{category}}, page {{page}} sur {{pageCount}}',
    loadedInCategory_other:
      '{{count}} articles publiés dans la catégorie {{category}}, page {{page}} sur {{pageCount}}',
    filter: {
      label: 'Catégorie',
      all: 'Toutes les catégories',
      unavailable: 'Les catégories sont momentanément indisponibles.',
    },
    showAllCategories: 'Voir toutes les catégories',
    empty: {
      title: 'Aucun article publié pour le moment',
      description: 'De nouveaux objets arrivent régulièrement : revenez bientôt.',
    },
    emptyCategory: {
      title: 'Aucun article publié dans cette catégorie',
      description: 'Choisissez une autre catégorie ou parcourez tout le catalogue.',
    },
    unknownCategory: {
      title: 'Cette catégorie n’existe pas',
      description: 'Le lien suivi mène à une catégorie inconnue du catalogue.',
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
  article: {
    documentTitle: '{{title}} · Collector',
    pendingDocumentTitle: 'Fiche article · Collector',
    pendingTitle: 'Fiche article',
    loading: 'Chargement de l’article en cours',
    loaded: 'Fiche de l’article {{title}} affichée',
    summaryTitle: 'En bref',
    categoryLink: 'Catégorie {{category}} : voir ses articles',
    shippingCost: 'Frais de port',
    shippingNotProvided: 'Non renseignés par le vendeur',
    seller: 'Vendeur',
    privateSeller: 'Vendeur particulier',
    descriptionTitle: 'Description',
    noDescription: 'Le vendeur n’a pas rédigé de description.',
    reserved: 'Réservé',
    reservation: {
      reserve: 'Réserver',
      hint: 'L’article vous est garanti le temps du paiement.',
      reservedReason: 'Cet article est déjà réservé : il n’est plus réservable.',
      signInPrompt: 'Connectez-vous avec un compte acheteur pour réserver cet article.',
      signIn: 'Connexion pour réserver',
      buyerOnly: 'Seul un compte acheteur peut réserver un article.',
      reconnect: 'Se reconnecter',
      failures: {
        alreadyReserved: 'Cet article est déjà réservé par un autre acheteur.',
        reservationAlreadyActive:
          'Vous avez déjà une réservation en cours sur un autre article. Réglez-la ou attendez son échéance avant d’en faire une nouvelle.',
        rateLimited: 'Trop de tentatives en peu de temps. Patientez un instant, puis réessayez.',
        notReservable: 'Cet article n’est plus réservable.',
        sessionExpired: 'Votre session a expiré. Reconnectez-vous, puis réessayez.',
        forbidden: 'Votre compte n’a pas le rôle acheteur : la réservation n’a pas été faite.',
        unavailable: 'La réservation a échoué. Vérifiez votre connexion, puis réessayez.',
      },
    },
    notFound: {
      documentTitle: 'Article introuvable · Collector',
      title: 'Article introuvable',
      description: 'Cet article n’existe pas ou n’est plus en vente.',
    },
    error: {
      title: 'L’article est momentanément indisponible',
      description: 'Vérifiez votre connexion, puis réessayez.',
      retry: 'Réessayer',
    },
  },
  photos: {
    unavailable: 'Photo momentanément indisponible',
    untitledAlt: 'Photo de l’article',
    galleryTitle: 'Photos de l’article',
    seller: {
      title: 'Photos',
      hint_one:
        'Ajoutez {{count}} photo de l’objet, en JPEG ou PNG : elle est exigée pour la publication. Les données de localisation sont retirées avant toute diffusion.',
      hint_other:
        'Ajoutez jusqu’à {{count}} photos de l’objet, en JPEG ou PNG : au moins une est exigée pour la publication. Les données de localisation sont retirées avant toute diffusion.',
      readOnly: 'Les photos ne se modifient que sur un brouillon ou une annonce refusée.',
      limitReached_one:
        'Une seule photo par article : retirez la photo actuelle pour la remplacer.',
      limitReached_other:
        '{{count}} photos au plus par article : retirez-en une pour en ajouter une autre.',
      remove: 'Retirer la photo',
      retry: 'Réessayer',
      loadFailed: 'Les photos sont momentanément indisponibles.',
    },
    upload: {
      dropTitle: 'Glissez-déposez une photo ici',
      dropDescription:
        'Ou choisissez un fichier. JPEG ou PNG, {{maxSize}} Mo et {{maxPixels}} mégapixels au plus.',
      choose: 'Choisir une photo',
      checking: 'Vérification du fichier…',
      progress: 'Envoi de la photo : {{percent}} %',
      previewAlt: 'Aperçu de la photo en cours d’envoi',
      cancel: 'Annuler l’envoi',
    },
    announcements: {
      uploading: 'Envoi de la photo en cours.',
      added: 'Photo ajoutée.',
      removed: 'Photo retirée.',
    },
    failures: {
      unsupportedType:
        'Ce fichier n’est pas une image JPEG ou PNG valide. Les formats GIF, WebP et SVG ne sont pas acceptés.',
      tooLarge: 'Fichier trop lourd : {{max}} Mo au plus.',
      tooManyPixels:
        'Image trop grande : {{max}} mégapixels au plus. Réduisez ses dimensions, puis réessayez.',
      limitReached_one: 'Cet article a déjà sa photo : retirez-la pour la remplacer.',
      limitReached_other: 'Cet article a déjà {{count}} photos, le maximum autorisé.',
      reserved: 'Cet article est réservé ou vendu : ses photos ne peuvent plus être modifiées.',
      locked: 'Les photos de cet article ne sont pas modifiables dans son état actuel.',
      rateLimited: 'Trop d’envois en peu de temps. Patientez un instant, puis réessayez.',
      busy: 'Le traitement des photos est saturé. Réessayez dans quelques instants.',
      notFound: 'Cet article ou cette photo n’existe plus.',
      unavailable: 'L’envoi a échoué. Vérifiez votre connexion, puis réessayez.',
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
      prix_abusif: 'Prix abusif',
      article_interdit: 'Article interdit à la vente',
      vendeur_non_fiable: 'Vendeur non fiable',
    },
  },
  admin: {
    documentTitle: 'Articles à vérifier · Collector',
    navigation: 'Articles à vérifier',
    review: {
      title: 'Articles à vérifier',
      subtitle:
        'Annonces que le contrôle automatique n’a pas pu trancher. Publiez-les ou refusez-les avec un motif.',
      listTitle: 'Annonces en attente de décision',
      listLabel: 'Annonces à vérifier',
      paginationLabel: 'Pagination des articles à vérifier',
      loading: 'Chargement des articles à vérifier en cours',
      loaded_one: '{{count}} article à vérifier',
      loaded_other: '{{count}} articles à vérifier',
      untitled: 'Article sans titre',
      notProvided: 'Non renseigné',
      controlReason: 'Motif du contrôle : {{reason}}',
      revision: 'Révision examinée : {{revision}}',
      facts: {
        description: 'Description',
        category: 'Catégorie',
        price: 'Prix',
        shippingCost: 'Frais de port',
        seller: 'Identifiant du vendeur',
        updatedAt: 'Modifié le',
      },
      actions: {
        publish: 'Publier',
        publishLabel: 'Publier « {{title}} »',
        refuse: 'Refuser',
        refuseLabel: 'Refuser « {{title}} »',
      },
      refusal: {
        title: 'Refuser l’annonce',
        description:
          '« {{title}} » ne sera pas publiée. Le vendeur verra l’état « Refusé » et le motif choisi.',
        reasonLabel: 'Motif du refus',
        missingReason: 'Choisissez un motif de refus.',
        cancel: 'Annuler',
        confirm: 'Confirmer le refus',
      },
      decided: {
        publie: '« {{title}} » a été publié.',
        refuse: '« {{title}} » a été refusé.',
      },
      conflict: {
        title: 'L’article a changé, rechargez la liste',
        description:
          'Il a été modifié ou déjà traité depuis son affichage : votre décision n’a pas été appliquée. Rechargez la liste, puis décidez de nouveau.',
        reload: 'Recharger la liste',
      },
      forbiddenDecision:
        'Votre compte n’a pas le rôle administrateur : la décision n’a pas été appliquée.',
      failure: 'La décision n’a pas pu être enregistrée. Vérifiez votre connexion, puis réessayez.',
      empty: {
        title: 'Aucun article à vérifier',
        description: 'Toutes les annonces ont été tranchées. Revenez plus tard.',
      },
      error: {
        title: 'Les articles à vérifier sont momentanément indisponibles',
        description: 'Vérifiez votre connexion, puis réessayez.',
        retry: 'Réessayer',
      },
      forbidden: {
        title: 'Accès réservé',
        description: 'Votre compte n’a pas le rôle requis pour ouvrir cet espace.',
      },
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
