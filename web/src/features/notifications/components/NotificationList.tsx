import Stack from '@mui/material/Stack';
import { revealSx } from '../../../ui/animations/revealSx';
import type { SellerNotification } from '../domain/sellerNotification';
import { NotificationCard } from './NotificationCard';

type NotificationListProps = {
  readonly notifications: readonly SellerNotification[];
};

export function NotificationList({ notifications }: NotificationListProps) {
  return (
    <Stack
      component="ul"
      spacing={{ xs: 1.5, md: 2 }}
      margin={0}
      padding={0}
      maxWidth="48rem"
      sx={{ listStyle: 'none' }}
    >
      {notifications.map((notification, index) => (
        <Stack component="li" key={notification.id} sx={revealSx(index)}>
          <NotificationCard notification={notification} />
        </Stack>
      ))}
    </Stack>
  );
}
