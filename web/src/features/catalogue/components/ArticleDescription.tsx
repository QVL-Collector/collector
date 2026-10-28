import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTranslation } from 'react-i18next';
import { revealSx } from '../../../ui/animations/revealSx';

type ArticleDescriptionProps = {
  readonly description: string | null;
};

export function ArticleDescription({ description }: ArticleDescriptionProps) {
  const { t } = useTranslation();
  return (
    <Stack
      component="section"
      aria-labelledby="fiche-description"
      spacing={1.5}
      flex={1}
      minWidth={0}
      sx={revealSx(1)}
    >
      <Typography id="fiche-description" component="h2" variant="h2">
        {t('article.descriptionTitle')}
      </Typography>
      {description === null ? (
        <Typography variant="body1" color="text.secondary">
          {t('article.noDescription')}
        </Typography>
      ) : (
        <Typography
          variant="body1"
          maxWidth="40rem"
          sx={{ whiteSpace: 'pre-line', overflowWrap: 'anywhere' }}
        >
          {description}
        </Typography>
      )}
    </Stack>
  );
}
