import ShoppingBagOutlinedIcon from '@mui/icons-material/ShoppingBagOutlined';
import { buyerSpacePath } from '../../features/orders/ordersPaths';
import { SpaceLink } from './SpaceLink';

type BuyerSpaceLinkProps = {
  readonly label: string;
  readonly compact: boolean;
};

export function BuyerSpaceLink({ label, compact }: BuyerSpaceLinkProps) {
  return (
    <SpaceLink
      to={buyerSpacePath}
      label={label}
      icon={<ShoppingBagOutlinedIcon />}
      compact={compact}
    />
  );
}
