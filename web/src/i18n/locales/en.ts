import type { Translation } from './fr';

export const en: Translation = {
  app: {
    name: 'Collector',
    homeLink: 'Collector, back to the catalogue',
    skipToContent: 'Skip to main content',
    language: {
      label: 'Interface language',
      frShort: 'FR',
      frFull: 'Français',
      enShort: 'EN',
      enFull: 'English',
    },
    bootstrapFailure: {
      title: 'The application could not start',
      description: 'Please reload the page in a moment.',
    },
  },
  auth: {
    signIn: 'Sign in',
    signOut: 'Sign out',
    accountLabel: 'Your account',
    unknownUser: 'Signed-in user',
    signInFailed: 'The sign-in service cannot be reached. Please try again in a moment.',
    signOutFailed:
      'The sign-in service could not confirm the sign-out. This browser has forgotten your session.',
    roles: {
      admin: 'Administrator',
      vendeur: 'Seller',
      acheteur: 'Buyer',
      none: 'No role',
    },
    callback: {
      documentTitle: 'Sign-in · Collector',
      pending: 'Signing you in…',
      title: 'Sign-in did not complete',
      description: 'Your session could not be opened. Start again with the “Sign in” button.',
    },
    gate: {
      redirecting: 'Redirecting to the sign-in page…',
      unavailableTitle: 'The sign-in service is unavailable',
      unavailableDescription: 'Check your connection, then try again.',
      retry: 'Try again',
    },
  },
  accessDenied: {
    documentTitle: 'Restricted access · Collector',
    title: 'Restricted access',
    description: 'Your account does not have the role required to open this area.',
  },
  seller: {
    documentTitle: 'Seller area · Collector',
    title: 'Seller area',
    welcome: 'Welcome, {{name}}. Your listings will appear here.',
    navigation: 'Seller area',
  },
  common: {
    backToCatalogue: 'Back to the catalogue',
    pagination: {
      page: 'Go to page {{page}}',
      currentPage: 'Page {{page}}, current page',
      first: 'First page',
      last: 'Last page',
      previous: 'Previous page',
      next: 'Next page',
    },
  },
  catalogue: {
    documentTitle: 'Catalogue · Collector',
    title: 'Catalogue',
    subtitle: 'Collectibles published by private sellers, checked before going online.',
    listLabel: 'Published items',
    paginationLabel: 'Catalogue pagination',
    loading: 'Loading the catalogue',
    loaded_one: '{{count}} published item, page {{page}} of {{pageCount}}',
    loaded_other: '{{count}} published items, page {{page}} of {{pageCount}}',
    empty: {
      title: 'No published items yet',
      description: 'New collectibles arrive regularly: come back soon.',
    },
    outOfRange: {
      title: 'This catalogue page is empty',
      description: 'The catalogue has fewer pages than the one requested.',
      action: 'Back to the first page',
    },
    error: {
      title: 'The catalogue is temporarily unavailable',
      description: 'Check your connection, then try again.',
      retry: 'Try again',
    },
  },
  control: {
    reasons: {
      annonce_incomplete: 'Incomplete listing',
      coordonnees_detectees: 'Contact details detected',
      compte_non_autorise: 'Account not allowed to sell',
      prix_a_confirmer: 'Price to be confirmed',
      vendeur_a_verifier: 'Seller to be reviewed',
      categorie_inconnue: 'Unknown category',
    },
  },
  notFound: {
    documentTitle: 'Page not found · Collector',
    title: 'Page not found',
    description: 'The requested address does not match any page.',
  },
  routeError: {
    documentTitle: 'Error · Collector',
    title: 'Something went wrong',
    description: 'This page could not be displayed.',
  },
};
