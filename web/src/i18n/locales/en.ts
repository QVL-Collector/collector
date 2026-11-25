import type { Translation } from './fr';

export const en: Translation = {
  app: {
    name: 'Collector',
    homeLink: 'Collector, back to the catalogue',
    skipToContent: 'Skip to main content',
    language: {
      label: 'Interface language',
      frShort: 'FR',
      frFull: 'FR, Français',
      enShort: 'EN',
      enFull: 'EN, English',
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
    untitled: 'Untitled item',
    newArticle: 'New item',
    backToList: 'Back to my items',
    states: {
      brouillon: 'Draft',
      en_controle: 'Under review',
      publie: 'Published',
      refuse: 'Rejected',
      a_verifier: 'To be checked',
    },
    stateDetails: {
      brouillon: 'Hidden from the catalogue until the listing is submitted and published.',
      en_controle: 'Automatic review in progress. This page updates itself as soon as it ends.',
      publie: 'The listing is visible in the catalogue.',
      refuse: 'Fix the listing, save it, then submit it again.',
      a_verifier:
        'An administrator must examine the listing. It stays hidden from the catalogue until then.',
    },
    reason: 'Reason: {{reason}}',
    announcements: {
      stateChanged: '“{{title}}”: {{state}}.',
      stateChangedWithReason: '“{{title}}”: {{state}}, reason: {{reason}}.',
    },
    list: {
      title: 'My items',
      paginationLabel: 'My items pagination',
      loading: 'Loading your items',
      loaded_one: '{{count}} item',
      loaded_other: '{{count}} items',
      updatedAt: 'Updated on {{date}}',
      noPrice: 'No price yet',
      empty: {
        title: 'No items yet',
        description: 'Create a draft: it stays hidden from the catalogue until it is published.',
      },
      error: {
        title: 'Your items are temporarily unavailable',
        description: 'Check your connection, then try again.',
        retry: 'Try again',
      },
    },
    editor: {
      newDocumentTitle: 'New item · Collector',
      newSubtitle:
        'Prepare your listing. It stays a draft, hidden from the catalogue, until you submit it.',
      documentTitle: '{{title}} · Collector',
      loading: 'Loading the item',
      notFound: {
        title: 'Item not found',
        description: 'This item does not exist or does not belong to your account.',
      },
      error: {
        title: 'The item is temporarily unavailable',
        description: 'Check your connection, then try again.',
        retry: 'Try again',
      },
    },
    form: {
      label: 'Listing content',
      hint: 'A draft may stay incomplete. Title, description, category and price are required for publication.',
      title: 'Title',
      description: 'Description',
      category: 'Category',
      noCategory: 'No category',
      categoriesUnavailable: 'Categories are temporarily unavailable.',
      price: 'Price (€)',
      shippingCost: 'Shipping cost (€)',
      amountHint: 'Example: 12.50',
      characterCount: '{{length}} of {{max}} characters',
      create: 'Create the draft',
      save: 'Save the draft',
      saved: 'Draft saved, revision {{revision}}.',
      saveForControl: 'Save and send back for review',
      savedForControl: 'Changes saved, revision {{revision}}: the listing is under review again.',
      recontrol: {
        published: {
          title: 'The listing will be removed from the catalogue',
          description:
            'Any saved change, even to the title alone, removes the listing from the catalogue during the review. It comes back once the new version is accepted.',
        },
        underReview: {
          title: 'The review will restart on the new version',
          description:
            'Any saved change restarts the review on your new version: the pending result for the current version will be ignored. The listing stays hidden from the catalogue until then.',
        },
      },
      invalid_one: 'The form contains {{count}} error: fix the highlighted field.',
      invalid_other: 'The form contains {{count}} errors: fix the highlighted fields.',
      locked:
        'The listing can no longer be edited in its current state. The page has been updated.',
      failure: 'Saving failed. Check your connection, then try again.',
      conflict: {
        title: 'This item was changed in the meantime',
        description:
          'It changed since you opened it (revision {{revision}}), for example in another tab. Your changes were not saved so that nothing is overwritten: reload the latest version, then make your changes again.',
        reload: 'Reload the latest version',
      },
      errors: {
        obligatoire: 'This field is required.',
        invalide: 'Invalid value. Amount example: 12.50.',
        hors_limites: 'Value outside the allowed limits.',
        trop_long: 'Text too long.',
        caractere_interdit: 'This text contains a character that is not allowed.',
        negatif: 'The amount cannot be negative.',
        au_dela_du_plafond: 'Amount above the allowed ceiling.',
        devise_non_prise_en_charge: 'Currency not supported.',
        categorie_inconnue: 'Unknown category.',
      },
    },
    submission: {
      title: 'Listing status',
      submit: 'Submit for review',
      unsavedChanges: 'Save your changes before submitting the listing.',
      polling: 'Review in progress, updating automatically…',
      conflict:
        'The item changed since it was displayed. It has just been reloaded: check it, then submit it again.',
      locked: 'The listing cannot be submitted in its current state. It has just been reloaded.',
      failure: 'Submission failed. Check your connection, then try again.',
    },
    reserved:
      'This item is reserved by a buyer or already sold: it cannot be changed or submitted during the reservation, nor after the sale.',
  },
  notifications: {
    documentTitle: 'Notifications · Collector',
    navigation: 'Notifications',
    title: 'Notifications',
    subtitle:
      'Every decision made on your items shows up here: publication, rejection or review by an administrator.',
    listTitle: 'Decisions on your items',
    loading: 'Loading your notifications',
    loaded_one: '{{count}} notification',
    loaded_other: '{{count}} notifications',
    decidedAt: 'Decided on {{date}}',
    revision: 'Revision {{revision}}',
    issues: {
      publie: {
        title: 'Your item has been published',
        description: 'It is now visible in the catalogue.',
      },
      refuse: {
        title: 'Your item has been rejected',
        description: 'Fix the listing, save it, then submit it again.',
      },
      a_verifier: {
        title: 'Your item needs to be checked',
        description:
          'An administrator will examine it. It stays hidden from the catalogue until then.',
      },
    },
    empty: {
      title: 'No notifications yet',
      description: 'You will be told here as soon as a decision is made on one of your items.',
    },
    error: {
      title: 'Your notifications are temporarily unavailable',
      description: 'Check your connection, then try again.',
      retry: 'Try again',
    },
    unauthenticated: {
      title: 'Your session has expired',
      description: 'Sign in again to see your notifications.',
      signIn: 'Sign in again',
    },
    forbidden: {
      title: 'Restricted access',
      description: 'Your account does not have the role required to see these notifications.',
    },
  },
  orders: {
    navigation: 'My reservations',
    list: {
      documentTitle: 'My reservations · Collector',
      title: 'My reservations',
      subtitle:
        'Find the items reserved in your name, their amount and how much time you have left to pay.',
      listTitle: 'Your reservations',
      listLabel: 'Reservations',
      paginationLabel: 'My reservations pagination',
      loading: 'Loading your reservations',
      loaded_one: '{{count}} reservation',
      loaded_other: '{{count}} reservations',
      total: 'Total amount',
      timeLeft: 'Expires {{relative}}',
      lastMinute: 'Expires in less than a minute',
      deadlineReached: 'Deadline reached, updating the status…',
      expired: 'The reservation of “{{title}}” has expired.',
      viewDetail: 'View the reservation',
      viewDetailLabel: 'View the reservation “{{title}}”',
      forbiddenDescription:
        'Your account does not have the role required to see these reservations.',
      empty: {
        title: 'No reservations yet',
        description: 'Reserve an item from the catalogue: it is held for you while you pay.',
        action: 'Browse the catalogue',
      },
      error: {
        title: 'Your reservations are temporarily unavailable',
        description: 'Check your connection, then try again.',
        retry: 'Try again',
      },
      unauthenticated: {
        description: 'Sign in again to see your reservations.',
      },
    },
    pendingDocumentTitle: 'Your order · Collector',
    pendingTitle: 'Your order',
    loading: 'Loading your order',
    views: {
      en_attente_paiement: {
        documentTitle: 'Item reserved · Collector',
        title: 'Item reserved',
        subtitle: 'Your order is recorded. The item is held for you until the payment deadline.',
        announcement: 'Reservation confirmed: order awaiting payment',
      },
      expiree: {
        documentTitle: 'Reservation expired · Collector',
        title: 'Reservation expired',
        subtitle: 'The reservation has expired, the item is available again.',
        announcement: 'Order expired: the reservation has expired, the item is available again',
      },
    },
    statusTitle: 'Order status',
    statuses: {
      en_attente_paiement: 'Awaiting payment',
      expiree: 'Expired',
    },
    createdAt: 'Ordered on {{date}}',
    deadlines: {
      en_attente_paiement: {
        title: 'Pay before',
        hint: 'Deadline set by the marketplace.',
      },
      expiree: {
        title: 'Deadline passed on',
        hint: 'The item is no longer held for you: reserve it again from its page if it is still available.',
      },
    },
    amountsTitle: 'Amounts',
    price: 'Reference price',
    shippingCost: 'Shipping',
    totalPaid: 'Total paid',
    splitTitle: 'How the total is split',
    commission: 'Collector commission (5% of the total)',
    sellerAmount: 'Amount owed to the seller',
    commissionHint:
      'The commission is taken from the total paid: it is not added to your payment. The seller receives the rest.',
    viewArticle: 'View the item',
    notFound: {
      documentTitle: 'Order not found · Collector',
      title: 'Order not found',
      description: 'This order does not exist or is not linked to your account.',
    },
    error: {
      title: 'Your order is temporarily unavailable',
      description: 'Check your connection, then try again.',
      retry: 'Try again',
    },
    unauthenticated: {
      title: 'Your session has expired',
      description: 'Sign in again to see your order.',
      signIn: 'Sign in again',
    },
    forbidden: {
      title: 'Restricted access',
      description: 'Your account does not have the role required to see this order.',
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
    loadedInCategory_one:
      '{{count}} published item in the {{category}} category, page {{page}} of {{pageCount}}',
    loadedInCategory_other:
      '{{count}} published items in the {{category}} category, page {{page}} of {{pageCount}}',
    filter: {
      label: 'Category',
      all: 'All categories',
      unavailable: 'Categories are temporarily unavailable.',
    },
    showAllCategories: 'Show all categories',
    empty: {
      title: 'No published items yet',
      description: 'New collectibles arrive regularly: come back soon.',
    },
    emptyCategory: {
      title: 'No published items in this category',
      description: 'Pick another category or browse the whole catalogue.',
    },
    unknownCategory: {
      title: 'This category does not exist',
      description: 'The link you followed points to a category unknown to the catalogue.',
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
  article: {
    documentTitle: '{{title}} · Collector',
    pendingDocumentTitle: 'Item details · Collector',
    pendingTitle: 'Item details',
    loading: 'Loading the item',
    loaded: 'Details of the item {{title}} displayed',
    summaryTitle: 'At a glance',
    categoryLink: 'Category {{category}}: see its items',
    shippingCost: 'Shipping',
    shippingNotProvided: 'Not provided by the seller',
    seller: 'Seller',
    privateSeller: 'Private seller',
    descriptionTitle: 'Description',
    noDescription: 'The seller has not written a description.',
    reserved: 'Reserved',
    reservation: {
      reserve: 'Reserve',
      hint: 'The item is held for you while you pay.',
      reservedReason: 'This item is already reserved: it can no longer be reserved.',
      signInPrompt: 'Sign in with a buyer account to reserve this item.',
      signIn: 'Reserve after signing in',
      buyerOnly: 'Only a buyer account can reserve an item.',
      reconnect: 'Sign in again',
      failures: {
        alreadyReserved: 'This item is already reserved by another buyer.',
        reservationAlreadyActive:
          'You already have an active reservation on another item. Pay for it or wait for its deadline before making a new one.',
        rateLimited: 'Too many attempts in a short time. Wait a moment, then try again.',
        notReservable: 'This item can no longer be reserved.',
        sessionExpired: 'Your session has expired. Sign in again, then try again.',
        forbidden: 'Your account does not have the buyer role: nothing was reserved.',
        unavailable: 'The reservation failed. Check your connection, then try again.',
      },
    },
    notFound: {
      documentTitle: 'Item not found · Collector',
      title: 'Item not found',
      description: 'This item does not exist or is no longer for sale.',
    },
    error: {
      title: 'The item is temporarily unavailable',
      description: 'Check your connection, then try again.',
      retry: 'Try again',
    },
  },
  photos: {
    unavailable: 'Photo temporarily unavailable',
    untitledAlt: 'Photo of the item',
    galleryTitle: 'Item photos',
    seller: {
      title: 'Photos',
      hint_one:
        'Add {{count}} photo of the item, as JPEG or PNG: it is required for publication. Location data is removed before anything is shown.',
      hint_other:
        'Add up to {{count}} photos of the item, as JPEG or PNG: at least one is required for publication. Location data is removed before anything is shown.',
      readOnly: 'Photos can only be changed on a draft or a refused listing.',
      limitReached_one: 'One photo per item: remove the current photo to replace it.',
      limitReached_other: '{{count}} photos at most per item: remove one to add another.',
      remove: 'Remove photo',
      retry: 'Try again',
      loadFailed: 'Photos are temporarily unavailable.',
    },
    upload: {
      dropTitle: 'Drag and drop a photo here',
      dropDescription:
        'Or choose a file. JPEG or PNG, at most {{maxSize}} MB and {{maxPixels}} megapixels.',
      choose: 'Choose a photo',
      checking: 'Checking the file…',
      progress: 'Uploading photo: {{percent}}%',
      previewAlt: 'Preview of the photo being uploaded',
      cancel: 'Cancel upload',
    },
    announcements: {
      uploading: 'Uploading photo.',
      added: 'Photo added.',
      removed: 'Photo removed.',
    },
    failures: {
      unsupportedType:
        'This file is not a valid JPEG or PNG image. GIF, WebP and SVG files are not accepted.',
      tooLarge: 'File too large: {{max}} MB at most.',
      tooManyPixels:
        'Image too large: {{max}} megapixels at most. Reduce its dimensions, then try again.',
      limitReached_one: 'This item already has its photo: remove it to replace it.',
      limitReached_other: 'This item already has {{count}} photos, the maximum allowed.',
      reserved: 'This item is reserved or sold: its photos can no longer be changed.',
      locked: 'This item’s photos cannot be changed in its current state.',
      rateLimited: 'Too many uploads in a short time. Wait a moment, then try again.',
      busy: 'Photo processing is saturated. Try again in a few moments.',
      notFound: 'This item or photo no longer exists.',
      unavailable: 'Upload failed. Check your connection, then try again.',
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
      prix_abusif: 'Excessive price',
      article_interdit: 'Item not allowed for sale',
      vendeur_non_fiable: 'Unreliable seller',
    },
  },
  admin: {
    documentTitle: 'Items to review · Collector',
    navigation: 'Items to review',
    review: {
      title: 'Items to review',
      subtitle:
        'Listings the automatic review could not decide. Publish them or reject them with a reason.',
      listTitle: 'Listings awaiting a decision',
      listLabel: 'Listings to review',
      paginationLabel: 'Items to review pagination',
      loading: 'Loading the items to review',
      loaded_one: '{{count}} item to review',
      loaded_other: '{{count}} items to review',
      untitled: 'Untitled item',
      notProvided: 'Not provided',
      controlReason: 'Review reason: {{reason}}',
      revision: 'Reviewed revision: {{revision}}',
      facts: {
        description: 'Description',
        category: 'Category',
        price: 'Price',
        shippingCost: 'Shipping cost',
        seller: 'Seller identifier',
        updatedAt: 'Updated on',
      },
      actions: {
        publish: 'Publish',
        publishLabel: 'Publish “{{title}}”',
        refuse: 'Reject',
        refuseLabel: 'Reject “{{title}}”',
      },
      refusal: {
        title: 'Reject the listing',
        description:
          '“{{title}}” will not be published. The seller will see the “Rejected” state and the chosen reason.',
        reasonLabel: 'Rejection reason',
        missingReason: 'Choose a rejection reason.',
        cancel: 'Cancel',
        confirm: 'Confirm the rejection',
      },
      decided: {
        publie: '“{{title}}” has been published.',
        refuse: '“{{title}}” has been rejected.',
      },
      conflict: {
        title: 'The item has changed, reload the list',
        description:
          'It was edited or already handled since it was displayed: your decision was not applied. Reload the list, then decide again.',
        reload: 'Reload the list',
      },
      forbiddenDecision:
        'Your account does not have the administrator role: the decision was not applied.',
      failure: 'The decision could not be saved. Check your connection, then try again.',
      empty: {
        title: 'No items to review',
        description: 'Every listing has been decided. Come back later.',
      },
      error: {
        title: 'The items to review are temporarily unavailable',
        description: 'Check your connection, then try again.',
        retry: 'Try again',
      },
      forbidden: {
        title: 'Restricted access',
        description: 'Your account does not have the role required to open this area.',
      },
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
