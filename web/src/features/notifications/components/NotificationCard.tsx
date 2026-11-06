import ChevronRightRounded from '@mui/icons-material/ChevronRightRounded';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { CardTitle } from '../../../ui/components/CardTitle';
import { LinkCard } from '../../../ui/components/LinkCard';
import type { SellerNotification } from '../domain/sellerNotification';
import { useNotificationCard } from '../hooks/useNotificationCard';
import { NotificationIssueChip } from './NotificationIssueChip';

type NotificationCardProps = {
  readonly notification: SellerNotification;
};

export function NotificationCard({ notification }: NotificationCardProps) {
  const card = useNotificationCard(notification);
  return (
    <LinkCard to={card.path}>
      <Stack direction="row" spacing={2} alignItems="center" padding={{ xs: 2, sm: 2.5 }}>
        <Stack spacing={1} flexGrow={1} minWidth={0}>
          <Stack
            direction={{ xs: 'column', sm: 'row' }}
            spacing={1}
            justifyContent="space-between"
            alignItems={{ xs: 'flex-start', sm: 'center' }}
          >
            <NotificationIssueChip issue={card.issue} label={card.issueLabel} />
            <Typography
              component="time"
              dateTime={card.decidedDateTime}
              variant="body2"
              color="text.secondary"
            >
              {card.decidedLabel}
            </Typography>
          </Stack>
          <CardTitle level="h3">{card.title}</CardTitle>
          <Typography variant="body2" color="text.secondary">
            {card.description}
          </Typography>
          {card.reasonLabel !== null && (
            <Typography component="p" variant="subtitle2">
              {card.reasonLabel}
            </Typography>
          )}
          <Typography variant="caption" color="text.secondary">
            {card.revisionLabel}
          </Typography>
        </Stack>
        <ChevronRightRounded color="action" />
      </Stack>
    </LinkCard>
  );
}
