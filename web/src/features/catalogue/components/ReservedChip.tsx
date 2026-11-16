import LockClockRounded from '@mui/icons-material/LockClockRounded';
import Chip from '@mui/material/Chip';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';

type ReservedChipProps = {
  readonly size: 'small' | 'medium';
};

export function ReservedChip({ size }: ReservedChipProps) {
  const { t } = useTranslation();
  return (
    <Chip
      icon={<LockClockRounded aria-hidden="true" />}
      label={t('article.reserved')}
      color="secondary"
      variant="filled"
      size={size}
      sx={revealSx(0)}
    />
  );
}
