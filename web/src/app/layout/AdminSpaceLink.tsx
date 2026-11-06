import FactCheckOutlinedIcon from '@mui/icons-material/FactCheckOutlined';
import { adminReviewPath } from '../../features/admin/adminPaths';
import { SpaceLink } from './SpaceLink';

type AdminSpaceLinkProps = {
  readonly label: string;
  readonly compact: boolean;
};

export function AdminSpaceLink({ label, compact }: AdminSpaceLinkProps) {
  return (
    <SpaceLink
      to={adminReviewPath}
      label={label}
      icon={<FactCheckOutlinedIcon />}
      compact={compact}
    />
  );
}
