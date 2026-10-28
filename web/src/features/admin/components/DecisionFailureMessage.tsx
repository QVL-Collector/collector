import RefreshRounded from '@mui/icons-material/RefreshRounded';
import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';
import type { DecisionFailure } from '../hooks/decisionFailure';

type DecisionFailureMessageProps = {
  readonly failure: DecisionFailure | null;
  readonly onReload: () => void;
};

export function DecisionFailureMessage({ failure, onReload }: DecisionFailureMessageProps) {
  const { t } = useTranslation();
  switch (failure) {
    case null:
      return null;
    case 'conflict':
      return (
        <Alert severity="error" sx={revealSx(0)}>
          <AlertTitle>{t('admin.review.conflict.title')}</AlertTitle>
          <Stack spacing={1.5} alignItems="flex-start">
            <Typography variant="body2">{t('admin.review.conflict.description')}</Typography>
            <Button
              variant="outlined"
              color="error"
              startIcon={<RefreshRounded />}
              onClick={onReload}
            >
              {t('admin.review.conflict.reload')}
            </Button>
          </Stack>
        </Alert>
      );
    case 'forbidden':
      return (
        <Alert severity="error" sx={revealSx(0)}>
          {t('admin.review.forbiddenDecision')}
        </Alert>
      );
    case 'unavailable':
      return (
        <Alert severity="error" sx={revealSx(0)}>
          {t('admin.review.failure')}
        </Alert>
      );
  }
}
