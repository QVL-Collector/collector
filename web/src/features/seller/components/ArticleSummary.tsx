import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';
import type { SellerArticle } from '../domain/sellerArticle';
import { useArticleSummary } from '../hooks/useArticleSummary';

type ArticleSummaryProps = {
  readonly article: SellerArticle;
};

export function ArticleSummary({ article }: ArticleSummaryProps) {
  const { t } = useTranslation();
  const entries = useArticleSummary(article);
  return (
    <Stack spacing={3}>
      <Typography variant="body2" color="text.secondary">
        {t('seller.editor.readOnly')}
      </Typography>
      <Stack component="dl" spacing={2} margin={0}>
        {entries.map(({ label, value }) => (
          <Stack key={label} spacing={0.5}>
            <Typography component="dt" variant="subtitle2" color="text.secondary">
              {label}
            </Typography>
            <Typography
              component="dd"
              variant="body1"
              margin={0}
              sx={{ whiteSpace: 'pre-line', overflowWrap: 'anywhere' }}
            >
              {value}
            </Typography>
          </Stack>
        ))}
      </Stack>
    </Stack>
  );
}
