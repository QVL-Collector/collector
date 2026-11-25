import AddPhotoAlternateRounded from '@mui/icons-material/AddPhotoAlternateRounded';
import { useTranslation } from 'react-i18next';
import { FileDropZone } from '../../../ui/components/FileDropZone';
import { photoMediaTypes } from '../domain/articlePhoto';
import { usePhotoLimitLabels } from '../hooks/usePhotoLimitLabels';

type PhotoDropAreaProps = {
  readonly onFiles: (files: readonly File[]) => void;
};

const acceptedTypes = photoMediaTypes.join(',');

export function PhotoDropArea({ onFiles }: PhotoDropAreaProps) {
  const { t } = useTranslation();
  const limits = usePhotoLimitLabels();
  return (
    <FileDropZone
      title={t('photos.upload.dropTitle')}
      description={t('photos.upload.dropDescription', {
        maxSize: limits.maxSize,
        maxPixels: limits.maxPixels,
      })}
      actionLabel={t('photos.upload.choose')}
      accept={acceptedTypes}
      icon={<AddPhotoAlternateRounded fontSize="inherit" />}
      onFiles={onFiles}
    />
  );
}
