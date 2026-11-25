import CloseRounded from '@mui/icons-material/CloseRounded';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import LinearProgress from '@mui/material/LinearProgress';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';
import type { PhotoUploadProgressView } from '../hooks/usePhotoUploadProgress';

type PhotoUploadProgressProps = {
  readonly progress: PhotoUploadProgressView;
  readonly previewUrl: string | null;
  readonly onCancel: () => void;
};

export function PhotoUploadProgress({ progress, previewUrl, onCancel }: PhotoUploadProgressProps) {
  const { t } = useTranslation();
  const labelId = useId();
  return (
    <Stack
      direction={{ xs: 'column', sm: 'row' }}
      spacing={2}
      alignItems={{ xs: 'stretch', sm: 'center' }}
      padding={2}
      border={1}
      borderColor="divider"
      borderRadius="1rem"
      bgcolor="background.paper"
      sx={revealSx(0)}
    >
      {previewUrl !== null && (
        <Box
          component="img"
          src={previewUrl}
          alt={t('photos.upload.previewAlt')}
          sx={{
            width: '4.5rem',
            height: '4.5rem',
            flexShrink: 0,
            objectFit: 'cover',
            borderRadius: '0.75rem',
          }}
        />
      )}
      <Stack spacing={1} flexGrow={1} minWidth={0}>
        <Typography id={labelId} variant="body2">
          {progress.label}
        </Typography>
        <LinearProgress
          variant={progress.isIndeterminate ? 'indeterminate' : 'determinate'}
          value={progress.percent}
          aria-labelledby={labelId}
          sx={{ height: '0.5rem', borderRadius: '62.5rem' }}
        />
      </Stack>
      {progress.canCancel && (
        <Button
          variant="text"
          startIcon={<CloseRounded />}
          onClick={onCancel}
          sx={{ minHeight: '2.75rem', alignSelf: { xs: 'flex-start', sm: 'center' } }}
        >
          {t('photos.upload.cancel')}
        </Button>
      )}
    </Stack>
  );
}
