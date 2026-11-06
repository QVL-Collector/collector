import NotificationsOutlinedIcon from '@mui/icons-material/NotificationsOutlined';
import { notificationsPath } from '../../features/notifications/notificationsPaths';
import { SpaceLink } from './SpaceLink';

type NotificationsSpaceLinkProps = {
  readonly label: string;
  readonly compact: boolean;
};

export function NotificationsSpaceLink({ label, compact }: NotificationsSpaceLinkProps) {
  return (
    <SpaceLink
      to={notificationsPath}
      label={label}
      icon={<NotificationsOutlinedIcon />}
      compact={compact}
    />
  );
}
