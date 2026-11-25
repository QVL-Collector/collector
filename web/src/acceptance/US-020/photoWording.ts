import type { SupportedLanguage } from '../../i18n/languages';

export type PhotoWording = {
  readonly section: string;
  readonly choose: string;
  readonly remove: string;
  readonly previewAlt: string;
  readonly added: string;
  readonly removed: string;
  readonly unsupported: string;
  readonly tooLarge: string;
  readonly tooManyPixels: string;
  readonly limitReached: string;
  readonly locked: string;
  readonly unavailable: string;
  readonly untitledAlt: string;
  readonly publishedSection: string;
};

export const photoWording: Readonly<Record<SupportedLanguage, PhotoWording>> = {
  fr: {
    section: 'Photos',
    choose: 'Choisir une photo',
    remove: 'Retirer la photo',
    previewAlt: 'Aperçu de la photo en cours d’envoi',
    added: 'Photo ajoutée.',
    removed: 'Photo retirée.',
    unsupported:
      'Ce fichier n’est pas une image JPEG ou PNG valide. Les formats GIF, WebP et SVG ne sont pas acceptés.',
    tooLarge: 'Fichier trop lourd : 5 Mo au plus.',
    tooManyPixels:
      'Image trop grande : 16 mégapixels au plus. Réduisez ses dimensions, puis réessayez.',
    limitReached: 'Cet article a déjà sa photo : retirez-la pour la remplacer.',
    locked: 'Les photos de cet article ne sont pas modifiables dans son état actuel.',
    unavailable: 'Photo momentanément indisponible',
    untitledAlt: 'Photo de l’article',
    publishedSection: 'Photos de l’article',
  },
  en: {
    section: 'Photos',
    choose: 'Choose a photo',
    remove: 'Remove photo',
    previewAlt: 'Preview of the photo being uploaded',
    added: 'Photo added.',
    removed: 'Photo removed.',
    unsupported:
      'This file is not a valid JPEG or PNG image. GIF, WebP and SVG files are not accepted.',
    tooLarge: 'File too large: 5 MB at most.',
    tooManyPixels: 'Image too large: 16 megapixels at most. Reduce its dimensions, then try again.',
    limitReached: 'This item already has its photo: remove it to replace it.',
    locked: 'This item’s photos cannot be changed in its current state.',
    unavailable: 'Photo temporarily unavailable',
    untitledAlt: 'Photo of the item',
    publishedSection: 'Item photos',
  },
};

export const fr = photoWording.fr;
