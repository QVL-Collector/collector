import Skeleton from '@mui/material/Skeleton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import { visuallyHidden } from '../../../ui/styles/visuallyHidden';
import type { PhotoGalleryView } from '../hooks/usePhotoGalleryView';
import { ArticlePhotoFigure } from './ArticlePhotoFigure';
import { PhotoGrid } from './PhotoGrid';

type ArticlePhotoGalleryProps = {
  readonly view: PhotoGalleryView;
  readonly headingComponent: 'h2' | 'h4';
  readonly maxWidth: string;
};

export function ArticlePhotoGallery({
  view,
  headingComponent,
  maxWidth,
}: ArticlePhotoGalleryProps) {
  const { t } = useTranslation();
  const headingId = useId();
  switch (view.status) {
    case 'hidden':
      return null;
    case 'loading':
      return (
        <Skeleton
          aria-hidden="true"
          variant="rounded"
          height="auto"
          sx={{ aspectRatio: '4 / 3', maxWidth }}
        />
      );
    case 'ready':
      return (
        <Stack component="section" aria-labelledby={headingId} maxWidth={maxWidth}>
          <Typography id={headingId} component={headingComponent} sx={visuallyHidden}>
            {t('photos.galleryTitle')}
          </Typography>
          <PhotoGrid maxWidth={maxWidth}>
            {view.items.map((item, index) => (
              <ArticlePhotoFigure
                key={item.photo.id}
                item={item}
                alt={view.altText}
                index={index}
                fit="contain"
                loading={index === 0 ? 'eager' : 'lazy'}
              />
            ))}
          </PhotoGrid>
        </Stack>
      );
  }
}
