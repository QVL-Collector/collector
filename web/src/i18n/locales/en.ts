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
