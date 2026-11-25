import { useCallback, useRef, useState, type RefObject } from 'react';
import { useTranslation } from 'react-i18next';
import type { ArticlePhoto } from '../domain/articlePhoto';
import { usePhotoAltText } from './usePhotoAltText';
import { usePhotoFailureMessage } from './usePhotoFailureMessage';
import { usePhotoLimits } from './usePhotoLimits';
import { usePhotoRemoval } from './usePhotoRemoval';
import { usePhotoUpload, type PhotoUploadModel } from './usePhotoUpload';
import type { PresignedPhotosModel } from './usePresignedPhotos';
import { useSellerArticlePhotos } from './useSellerArticlePhotos';

export type SellerPhotoSectionOptions = {
  readonly articleId: string;
  readonly articleTitle: string;
  readonly canEdit: boolean;
};

export type SellerPhotoSectionModel = {
  readonly isAvailable: boolean;
  readonly headingRef: RefObject<HTMLHeadingElement | null>;
  readonly gallery: PresignedPhotosModel;
  readonly altText: string;
  readonly canEdit: boolean;
  readonly canAddPhoto: boolean;
  readonly isFull: boolean;
  readonly maxPerArticle: number;
  readonly upload: PhotoUploadModel;
  readonly uploadFailure: string | null;
  readonly removingPhotoId: string | null;
  readonly removalFailure: string | null;
  readonly removePhoto: ((photo: ArticlePhoto) => void) | null;
  readonly announcement: string;
};

export function useSellerPhotoSection({
  articleId,
  articleTitle,
  canEdit,
}: SellerPhotoSectionOptions): SellerPhotoSectionModel {
  const { t } = useTranslation();
  const headingRef = useRef<HTMLHeadingElement | null>(null);
  const [lastEvent, setLastEvent] = useState('');
  const gallery = useSellerArticlePhotos(articleId);
  const altText = usePhotoAltText(articleTitle);

  const onUploaded = useCallback(() => {
    setLastEvent(t('photos.announcements.added'));
  }, [t]);

  const onRemoved = useCallback(() => {
    setLastEvent(t('photos.announcements.removed'));
    headingRef.current?.focus();
  }, [t]);

  const upload = usePhotoUpload({ articleId, onUploaded });
  const removal = usePhotoRemoval(articleId, onRemoved);
  const uploadFailure = usePhotoFailureMessage(
    upload.phase.kind === 'failed' ? upload.phase.failure : null,
  );
  const removalFailure = usePhotoFailureMessage(removal.failure);
  const { maxPerArticle } = usePhotoLimits();
  const isReady = gallery.status === 'ready';
  const isFull = isReady && gallery.count >= maxPerArticle;

  return {
    isAvailable: gallery.status !== 'disabled',
    headingRef,
    gallery,
    altText,
    canEdit,
    canAddPhoto: canEdit && isReady && !isFull,
    isFull,
    maxPerArticle,
    upload,
    uploadFailure,
    removingPhotoId: removal.removingPhotoId,
    removalFailure,
    removePhoto: canEdit ? removal.removePhoto : null,
    announcement: upload.isBusy ? t('photos.announcements.uploading') : lastEvent,
  };
}
