import DeleteOutlineRounded from '@mui/icons-material/DeleteOutlineRounded';
import Button from '@mui/material/Button';
import { useTranslation } from 'react-i18next';

type RemovePhotoButtonProps = {
  readonly isRemoving: boolean;
  readonly disabled: boolean;
  readonly onRemove: () => void;
};

export function RemovePhotoButton({ isRemoving, disabled, onRemove }: RemovePhotoButtonProps) {
  const { t } = useTranslation();
  return (
    <Button
      variant="text"
      color="error"
      startIcon={<DeleteOutlineRounded />}
      loading={isRemoving}
      loadingPosition="start"
      disabled={disabled}
      onClick={onRemove}
      sx={{ alignSelf: 'flex-start', minHeight: '2.75rem' }}
    >
      {t('photos.seller.remove')}
    </Button>
  );
}
