import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import { LiveStatus } from '../../../ui/components/LiveStatus';
import {
  useSellerPhotoSection,
  type SellerPhotoSectionOptions,
} from '../hooks/useSellerPhotoSection';
import { PhotoUploadPanel } from './PhotoUploadPanel';
import { SellerPhotoGallery } from './SellerPhotoGallery';

export function SellerPhotoSection(options: SellerPhotoSectionOptions) {
  const { t } = useTranslation();
  const headingId = useId();
  const { isAvailable, headingRef, gallery, altText, canEdit, announcement, ...panel } =
    useSellerPhotoSection(options);
  if (!isAvailable) {
    return null;
  }
  return (
    <Paper
      component="section"
      variant="outlined"
      aria-labelledby={headingId}
      sx={{ borderRadius: '1rem' }}
    >
      <Stack spacing={2.5} padding={{ xs: 2.5, md: 3 }}>
        <Stack spacing={0.5}>
          <Typography ref={headingRef} id={headingId} component="h2" variant="h6" tabIndex={-1}>
            {t('photos.seller.title')}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {canEdit
              ? t('photos.seller.hint', { count: panel.maxPerArticle })
              : t('photos.seller.readOnly')}
          </Typography>
        </Stack>
        <SellerPhotoGallery
          gallery={gallery}
          altText={altText}
          removingPhotoId={panel.removingPhotoId}
          isUploading={panel.upload.isBusy}
          onRemove={panel.removePhoto}
        />
        {canEdit && <PhotoUploadPanel {...panel} />}
        <LiveStatus message={announcement} />
      </Stack>
    </Paper>
  );
}
