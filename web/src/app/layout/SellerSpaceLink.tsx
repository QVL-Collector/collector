import StorefrontOutlinedIcon from '@mui/icons-material/StorefrontOutlined';
import { sellerSpacePath } from '../../features/seller/sellerPaths';
import { SpaceLink } from './SpaceLink';

type SellerSpaceLinkProps = {
  readonly label: string;
  readonly compact: boolean;
};

export function SellerSpaceLink({ label, compact }: SellerSpaceLinkProps) {
  return (
    <SpaceLink
      to={sellerSpacePath}
      label={label}
      icon={<StorefrontOutlinedIcon />}
      compact={compact}
    />
  );
}
