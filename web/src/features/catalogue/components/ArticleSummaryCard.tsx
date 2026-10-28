import LocalShippingOutlined from '@mui/icons-material/LocalShippingOutlined';
import PersonOutlineRounded from '@mui/icons-material/PersonOutlineRounded';
import Card from '@mui/material/Card';
import Chip from '@mui/material/Chip';
import Divider from '@mui/material/Divider';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';
import { Link as RouterLink } from 'react-router';
import { revealSx } from '../../../ui/animations/revealSx';
import { FactItem } from '../../../ui/components/FactItem';
import { FactList } from '../../../ui/components/FactList';
import type { PublishedArticleSummary } from '../hooks/usePublishedArticleSummary';

type ArticleSummaryCardProps = {
  readonly summary: PublishedArticleSummary;
};

export function ArticleSummaryCard({ summary }: ArticleSummaryCardProps) {
  const { t } = useTranslation();
  return (
    <Card component="section" variant="outlined" aria-labelledby="fiche-achat" sx={revealSx(0)}>
      <Stack spacing={2.5} padding={{ xs: 2.5, md: 3 }}>
        <Stack spacing={1.5} alignItems="flex-start">
          <Typography id="fiche-achat" component="h2" variant="overline" color="text.secondary">
            {t('article.summaryTitle')}
          </Typography>
          <Chip
            component={RouterLink}
            to={summary.categoryPath}
            clickable
            label={summary.categoryLabel}
            aria-label={t('article.categoryLink', { category: summary.categoryLabel })}
            variant="outlined"
            color="primary"
            sx={{ height: '2.75rem', borderRadius: '62.5rem' }}
          />
          <Typography component="p" variant="h2" color="primary">
            {summary.priceLabel}
          </Typography>
        </Stack>
        <Divider />
        <FactList>
          <FactItem
            label={t('article.shippingCost')}
            icon={<LocalShippingOutlined fontSize="small" />}
          >
            {summary.shippingLabel}
          </FactItem>
          <FactItem label={t('article.seller')} icon={<PersonOutlineRounded fontSize="small" />}>
            {t('article.privateSeller')}
          </FactItem>
        </FactList>
      </Stack>
    </Card>
  );
}
