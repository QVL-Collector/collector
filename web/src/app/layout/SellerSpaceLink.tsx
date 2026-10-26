import StorefrontOutlinedIcon from '@mui/icons-material/StorefrontOutlined';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import { Link as RouterLink } from 'react-router';
import { sellerSpacePath } from '../../features/seller/sellerPaths';

type SellerSpaceLinkProps = {
  readonly label: string;
  readonly compact: boolean;
};

export function SellerSpaceLink({ label, compact }: SellerSpaceLinkProps) {
  if (compact) {
    return (
      <IconButton
        component={RouterLink}
        to={sellerSpacePath}
        aria-label={label}
        color="primary"
        size="large"
      >
        <StorefrontOutlinedIcon />
      </IconButton>
    );
  }
  return (
    <Button
      component={RouterLink}
      to={sellerSpacePath}
      variant="text"
      startIcon={<StorefrontOutlinedIcon />}
      sx={{ minHeight: '2.75rem' }}
    >
      {label}
    </Button>
  );
}
