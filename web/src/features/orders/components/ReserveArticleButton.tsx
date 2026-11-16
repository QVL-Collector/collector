import BookmarkAddedRounded from '@mui/icons-material/BookmarkAddedRounded';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import type { ReservationFailure } from '../hooks/reservationFailure';
import { useReserveArticle } from '../hooks/useReserveArticle';

type ReserveArticleButtonProps = {
  readonly articleId: string;
  readonly blocked: boolean;
  readonly onStart: () => void;
  readonly onFailure: (failure: ReservationFailure) => void;
};

export function ReserveArticleButton({
  articleId,
  blocked,
  onStart,
  onFailure,
}: ReserveArticleButtonProps) {
  const { t } = useTranslation();
  const hintId = useId();
  const { isReserving, reserve } = useReserveArticle({ articleId, onStart, onFailure });
  return (
    <Stack spacing={1}>
      <Button
        variant="contained"
        size="large"
        fullWidth
        startIcon={<BookmarkAddedRounded />}
        loading={isReserving}
        loadingPosition="start"
        disabled={blocked}
        aria-describedby={hintId}
        onClick={reserve}
        sx={{
          minHeight: '3rem',
          transition: (theme) => theme.transitions.create(['transform', 'box-shadow']),
          '&:hover': { transform: 'translateY(-0.0625rem)', boxShadow: 2 },
          '&:active': { transform: 'scale(0.98)' },
          '@media (prefers-reduced-motion: reduce)': {
            '&:hover, &:active': { transform: 'none' },
          },
        }}
      >
        {t('article.reservation.reserve')}
      </Button>
      <Typography id={hintId} variant="body2" color="text.secondary">
        {t('article.reservation.hint')}
      </Typography>
    </Stack>
  );
}
