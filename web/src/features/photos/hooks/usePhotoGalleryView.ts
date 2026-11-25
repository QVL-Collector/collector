import { usePhotoAltText } from './usePhotoAltText';
import type { PhotoDisplayItem, PresignedPhotosModel } from './usePresignedPhotos';

export type PhotoGalleryView =
  | { readonly status: 'hidden' }
  | { readonly status: 'loading' }
  | {
      readonly status: 'ready';
      readonly items: readonly PhotoDisplayItem[];
      readonly altText: string;
    };

export function usePhotoGalleryView(
  gallery: PresignedPhotosModel,
  articleTitle: string,
): PhotoGalleryView {
  const altText = usePhotoAltText(articleTitle);
  switch (gallery.status) {
    case 'disabled':
    case 'error':
      return { status: 'hidden' };
    case 'loading':
      return { status: 'loading' };
    case 'ready':
      return gallery.count === 0
        ? { status: 'hidden' }
        : { status: 'ready', items: gallery.items, altText };
  }
}
