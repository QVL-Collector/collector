import SendRounded from '@mui/icons-material/SendRounded';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';
import type { SubmissionFailure } from '../hooks/useSubmitForControl';
import { SubmissionFailureMessage } from './SubmissionFailureMessage';

type ArticleSubmitActionProps = {
  readonly isSubmitting: boolean;
  readonly isBlocked: boolean;
  readonly failure: SubmissionFailure | null;
  readonly onSubmit: () => void;
};

export function ArticleSubmitAction({
  isSubmitting,
  isBlocked,
  failure,
  onSubmit,
}: ArticleSubmitActionProps) {
  const { t } = useTranslation();
  return (
    <Stack spacing={1.5}>
      <Button
        variant="contained"
        startIcon={<SendRounded />}
        loading={isSubmitting}
        loadingPosition="start"
        disabled={isBlocked}
        onClick={onSubmit}
        sx={{ minHeight: '2.75rem' }}
      >
        {t('seller.submission.submit')}
      </Button>
      {isBlocked && (
        <Typography variant="body2" color="text.secondary">
          {t('seller.submission.unsavedChanges')}
        </Typography>
      )}
      {failure !== null && (
        <Alert severity="error" sx={revealSx(0)}>
          <SubmissionFailureMessage failure={failure} />
        </Alert>
      )}
    </Stack>
  );
}
