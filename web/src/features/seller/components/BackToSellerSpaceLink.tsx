import ArrowBackRounded from '@mui/icons-material/ArrowBackRounded';
import Stack from '@mui/material/Stack';
import { useTranslation } from 'react-i18next';
import { LinkButton } from '../../../ui/components/LinkButton';
import { sellerSpacePath } from '../sellerPaths';

export function BackToSellerSpaceLink() {
  const { t } = useTranslation();
  return (
    <Stack alignItems="flex-start">
      <LinkButton
        to={sellerSpacePath}
        label={t('seller.backToList')}
        icon={<ArrowBackRounded />}
        variant="text"
      />
    </Stack>
  );
}
