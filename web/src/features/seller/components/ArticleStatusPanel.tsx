import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useId, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import type { ArticleStatus } from '../hooks/useArticleStatus';
import { ArticleStateChip } from './ArticleStateChip';
import { ControlInProgress } from './ControlInProgress';

type ArticleStatusPanelProps = {
  readonly status: ArticleStatus;
  readonly action?: ReactNode;
};

export function ArticleStatusPanel({ status, action }: ArticleStatusPanelProps) {
  const { t } = useTranslation();
  const headingId = useId();
  return (
    <Paper
      component="section"
      variant="outlined"
      aria-labelledby={headingId}
      sx={{ borderRadius: '1rem' }}
    >
      <Stack spacing={2} padding={{ xs: 2.5, md: 3 }} alignItems="flex-start">
        <Typography id={headingId} component="h2" variant="h6">
          {t('seller.submission.title')}
        </Typography>
        <ArticleStateChip state={status.state} label={status.label} />
        {status.reason !== null && (
          <Typography variant="subtitle1" component="p">
            {status.reason}
          </Typography>
        )}
        <Typography variant="body2" color="text.secondary">
          {status.detail}
        </Typography>
        {status.isUnderControl && (
          <Stack alignSelf="stretch">
            <ControlInProgress />
          </Stack>
        )}
        {action !== undefined && <Stack alignSelf="stretch">{action}</Stack>}
      </Stack>
    </Paper>
  );
}
