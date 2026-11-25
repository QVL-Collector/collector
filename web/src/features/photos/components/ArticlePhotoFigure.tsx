import Stack from '@mui/material/Stack';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';
import { ImageFrame, type ImageFit } from '../../../ui/components/ImageFrame';
import type { PhotoDisplayItem } from '../hooks/usePresignedPhotos';

type ArticlePhotoFigureProps = {
  readonly item: PhotoDisplayItem;
  readonly alt: string;
  readonly index: number;
  readonly fit: ImageFit;
  readonly loading?: 'lazy' | 'eager';
  readonly action?: ReactNode;
};

export function ArticlePhotoFigure({
  item,
  alt,
  index,
  fit,
  loading,
  action,
}: ArticlePhotoFigureProps) {
  const { t } = useTranslation();
  return (
    <Stack component="figure" margin={0} spacing={1} sx={revealSx(index)}>
      <ImageFrame
        src={item.photo.url}
        alt={alt}
        fit={fit}
        isBroken={item.isBroken}
        unavailableLabel={t('photos.unavailable')}
        loading={loading}
        onError={item.onError}
        onLoad={item.onLoad}
      />
      {action}
    </Stack>
  );
}
