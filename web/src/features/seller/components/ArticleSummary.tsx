import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';
import { FactItem } from '../../../ui/components/FactItem';
import { FactList } from '../../../ui/components/FactList';
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
      <FactList>
        {entries.map(({ label, value }) => (
          <FactItem key={label} label={label}>
            {value}
          </FactItem>
        ))}
      </FactList>
    </Stack>
  );
}
