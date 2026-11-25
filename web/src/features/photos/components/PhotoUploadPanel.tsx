import Alert from '@mui/material/Alert';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';
import { usePhotoUploadProgress } from '../hooks/usePhotoUploadProgress';
import type { SellerPhotoSectionModel } from '../hooks/useSellerPhotoSection';
import { PhotoDropArea } from './PhotoDropArea';
import { PhotoUploadProgress } from './PhotoUploadProgress';

type PhotoUploadPanelProps = Pick<
  SellerPhotoSectionModel,
  'upload' | 'uploadFailure' | 'removalFailure' | 'canAddPhoto' | 'isFull' | 'maxPerArticle'
>;

export function PhotoUploadPanel({
  upload,
  uploadFailure,
  removalFailure,
  canAddPhoto,
  isFull,
  maxPerArticle,
}: PhotoUploadPanelProps) {
  const { t } = useTranslation();
  const progress = usePhotoUploadProgress(upload.phase);
  return (
    <Stack spacing={2}>
      {progress !== null && (
        <PhotoUploadProgress
          progress={progress}
          previewUrl={upload.previewUrl}
          onCancel={upload.cancel}
        />
      )}
      {progress === null && canAddPhoto && <PhotoDropArea onFiles={upload.selectFiles} />}
      {progress === null && isFull && (
        <Typography variant="body2" color="text.secondary">
          {t('photos.seller.limitReached', { count: maxPerArticle })}
        </Typography>
      )}
      {uploadFailure !== null && (
        <Alert severity="error" onClose={upload.dismissFailure} sx={revealSx(0)}>
          {uploadFailure}
        </Alert>
      )}
      {removalFailure !== null && (
        <Alert severity="error" sx={revealSx(0)}>
          {removalFailure}
        </Alert>
      )}
    </Stack>
  );
}
