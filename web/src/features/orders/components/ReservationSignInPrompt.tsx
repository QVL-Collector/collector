import LoginRounded from '@mui/icons-material/LoginRounded';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId } from 'react';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';

type ReservationSignInPromptProps = {
  readonly signInFailed: boolean;
  readonly onSignIn: () => void;
};

export function ReservationSignInPrompt({ signInFailed, onSignIn }: ReservationSignInPromptProps) {
  const { t } = useTranslation();
  const promptId = useId();
  return (
    <Stack spacing={1}>
      <Typography id={promptId} variant="body2" color="text.secondary">
        {t('article.reservation.signInPrompt')}
      </Typography>
      <Button
        variant="outlined"
        size="large"
        fullWidth
        startIcon={<LoginRounded />}
        aria-describedby={promptId}
        onClick={onSignIn}
        sx={{ minHeight: '3rem' }}
      >
        {t('article.reservation.signIn')}
      </Button>
      {signInFailed && (
        <Alert severity="error" sx={revealSx(0)}>
          {t('auth.signInFailed')}
        </Alert>
      )}
    </Stack>
  );
}
