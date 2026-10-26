import CheckCircleRounded from '@mui/icons-material/CheckCircleRounded';
import RefreshRounded from '@mui/icons-material/RefreshRounded';
import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';
import type { FormFeedback } from '../hooks/useArticleEditor';

type ArticleFormFeedbackProps = {
  readonly feedback: FormFeedback;
  readonly onReload: () => void;
};

export function ArticleFormFeedback({ feedback, onReload }: ArticleFormFeedbackProps) {
  const { t } = useTranslation();
  switch (feedback.kind) {
    case 'none':
      return null;
    case 'saved':
      return (
        <Stack
          direction="row"
          spacing={1}
          alignItems="center"
          color="primary.main"
          sx={revealSx(0)}
        >
          <CheckCircleRounded fontSize="small" />
          <Typography variant="body2">
            {t('seller.form.saved', { revision: feedback.revision })}
          </Typography>
        </Stack>
      );
    case 'invalid':
      return (
        <Alert severity="error" sx={revealSx(0)}>
          {t('seller.form.invalid', { count: feedback.count })}
        </Alert>
      );
    case 'conflict':
      return (
        <Alert severity="error" sx={revealSx(0)}>
          <AlertTitle>{t('seller.form.conflict.title')}</AlertTitle>
          <Stack spacing={1.5} alignItems="flex-start">
            <Typography variant="body2">
              {t('seller.form.conflict.description', { revision: feedback.revision })}
            </Typography>
            <Button
              variant="outlined"
              color="error"
              startIcon={<RefreshRounded />}
              onClick={onReload}
            >
              {t('seller.form.conflict.reload')}
            </Button>
          </Stack>
        </Alert>
      );
    case 'locked':
      return (
        <Alert severity="error" sx={revealSx(0)}>
          {t('seller.form.locked')}
        </Alert>
      );
    case 'unavailable':
      return (
        <Alert severity="error" sx={revealSx(0)}>
          {t('seller.form.failure')}
        </Alert>
      );
  }
}
