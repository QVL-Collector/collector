import CheckCircleRounded from '@mui/icons-material/CheckCircleRounded';
import HighlightOffRounded from '@mui/icons-material/HighlightOffRounded';
import ManageSearchRounded from '@mui/icons-material/ManageSearchRounded';
import Chip, { type ChipProps } from '@mui/material/Chip';
import type { ReactElement } from 'react';
import type { NotificationIssue } from '../domain/sellerNotification';

type IssueAppearance = {
  readonly icon: ReactElement;
  readonly color: ChipProps['color'];
  readonly variant: ChipProps['variant'];
};

function appearanceOf(issue: NotificationIssue): IssueAppearance {
  switch (issue) {
    case 'publie':
      return { icon: <CheckCircleRounded />, color: 'primary', variant: 'filled' };
    case 'refuse':
      return { icon: <HighlightOffRounded />, color: 'error', variant: 'outlined' };
    case 'a_verifier':
      return { icon: <ManageSearchRounded />, color: 'secondary', variant: 'outlined' };
  }
}

type NotificationIssueChipProps = {
  readonly issue: NotificationIssue;
  readonly label: string;
};

export function NotificationIssueChip({ issue, label }: NotificationIssueChipProps) {
  const { icon, color, variant } = appearanceOf(issue);
  return <Chip icon={icon} label={label} color={color} variant={variant} size="small" />;
}
