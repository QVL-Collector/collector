import BookmarkAddedRounded from '@mui/icons-material/BookmarkAddedRounded';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';

export function ReservationUnavailable() {
  const { t } = useTranslation();
  const reasonId = useId();
  return (
    <Stack spacing={1}>
      <Button
        variant="contained"
        size="large"
        fullWidth
        startIcon={<BookmarkAddedRounded />}
        disabled
        aria-describedby={reasonId}
        sx={{ minHeight: '3rem' }}
      >
        {t('article.reservation.reserve')}
      </Button>
      <Typography id={reasonId} variant="body2" color="text.secondary">
        {t('article.reservation.reservedReason')}
      </Typography>
    </Stack>
  );
}
