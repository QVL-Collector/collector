import { useTranslation } from 'react-i18next';
import type { PhotoUploadPhase } from './usePhotoUpload';

export type PhotoUploadProgressView = {
  readonly label: string;
  readonly percent: number;
  readonly isIndeterminate: boolean;
  readonly canCancel: boolean;
};

const fullPercent = 100;

export function usePhotoUploadProgress(phase: PhotoUploadPhase): PhotoUploadProgressView | null {
  const { t } = useTranslation();
  switch (phase.kind) {
    case 'idle':
    case 'failed':
      return null;
    case 'checking':
      return {
        label: t('photos.upload.checking'),
        percent: 0,
        isIndeterminate: true,
        canCancel: false,
      };
    case 'uploading': {
      const percent = Math.round(Math.min(Math.max(phase.progress, 0), 1) * fullPercent);
      return {
        label: t('photos.upload.progress', { percent }),
        percent,
        isIndeterminate: false,
        canCancel: true,
      };
    }
  }
}
