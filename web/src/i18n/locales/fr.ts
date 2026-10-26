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
