import RefreshRounded from '@mui/icons-material/RefreshRounded';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Skeleton from '@mui/material/Skeleton';
import { useTranslation } from 'react-i18next';
import type { ArticlePhoto } from '../domain/articlePhoto';
import type { PresignedPhotosModel } from '../hooks/usePresignedPhotos';
import { ArticlePhotoFigure } from './ArticlePhotoFigure';
import { PhotoGrid } from './PhotoGrid';
import { RemovePhotoButton } from './RemovePhotoButton';

type SellerPhotoGalleryProps = {
  readonly gallery: PresignedPhotosModel;
  readonly altText: string;
  readonly removingPhotoId: string | null;
  readonly isUploading: boolean;
  readonly onRemove: ((photo: ArticlePhoto) => void) | null;
};

export function SellerPhotoGallery({
  gallery,
  altText,
  removingPhotoId,
  isUploading,
  onRemove,
}: SellerPhotoGalleryProps) {
  const { t } = useTranslation();
  switch (gallery.status) {
    case 'disabled':
      return null;
    case 'loading':
      return (
        <PhotoGrid>
          <Skeleton
            aria-hidden="true"
            variant="rounded"
            height="auto"
            sx={{ aspectRatio: '4 / 3' }}
          />
        </PhotoGrid>
      );
    case 'error':
      return (
        <Alert
          severity="error"
          role="status"
          action={
            <Button
              color="inherit"
              size="small"
              startIcon={<RefreshRounded />}
              onClick={gallery.retry}
            >
              {t('photos.seller.retry')}
            </Button>
          }
        >
          {t('photos.seller.loadFailed')}
        </Alert>
      );
    case 'ready':
      return gallery.count === 0 ? null : (
        <PhotoGrid>
          {gallery.items.map((item, index) => (
            <ArticlePhotoFigure
              key={item.photo.id}
              item={item}
              alt={altText}
              index={index}
              fit="cover"
              action={
                onRemove === null ? undefined : (
                  <RemovePhotoButton
                    isRemoving={removingPhotoId === item.photo.id}
                    disabled={isUploading || removingPhotoId !== null}
                    onRemove={() => {
                      onRemove(item.photo);
                    }}
                  />
                )
              }
            />
          ))}
        </PhotoGrid>
      );
  }
}
