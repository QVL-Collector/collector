import Alert from '@mui/material/Alert';
import AlertTitle from '@mui/material/AlertTitle';
import { revealSx } from '../../../ui/animations/revealSx';
import type { RecontrolWarning } from '../hooks/useRecontrolWarning';

type RecontrolWarningNoticeProps = {
  readonly id: string;
  readonly warning: RecontrolWarning;
};

export function RecontrolWarningNotice({ id, warning }: RecontrolWarningNoticeProps) {
  return (
    <Alert id={id} severity="warning" role="note" sx={revealSx(0)}>
      <AlertTitle>{warning.title}</AlertTitle>
      {warning.description}
    </Alert>
  );
}
