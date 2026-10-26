import LinearProgress from '@mui/material/LinearProgress';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';

export function ControlInProgress() {
  const { t } = useTranslation();
  return (
    <Stack spacing={1}>
      <LinearProgress
        aria-hidden="true"
        sx={{
          borderRadius: '62.5rem',
          '@media (prefers-reduced-motion: reduce)': { display: 'none' },
        }}
      />
      <Typography variant="body2" color="text.secondary">
        {t('seller.submission.polling')}
      </Typography>
    </Stack>
  );
}
