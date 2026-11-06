import BlockRounded from '@mui/icons-material/BlockRounded';
import PublishRounded from '@mui/icons-material/PublishRounded';
import Button from '@mui/material/Button';
import Stack from '@mui/material/Stack';
import type { ReviewArticleCardLabels } from '../hooks/useReviewArticleCard';
import type { PendingDecision } from '../hooks/useReviewDecision';

type ReviewDecisionActionsProps = {
  readonly labels: ReviewArticleCardLabels;
  readonly pendingDecision: PendingDecision;
  readonly canDecide: boolean;
  readonly onPublish: () => void;
  readonly onRefuse: () => void;
};

const touchTargetSx = { minHeight: '2.75rem' } as const;

export function ReviewDecisionActions({
  labels,
  pendingDecision,
  canDecide,
  onPublish,
  onRefuse,
}: ReviewDecisionActionsProps) {
  return (
    <Stack direction={{ xs: 'column-reverse', sm: 'row' }} spacing={1.5} justifyContent="flex-end">
      <Button
        variant="outlined"
        color="error"
        startIcon={<BlockRounded />}
        aria-label={labels.refuseAccessible}
        loading={pendingDecision === 'refuse'}
        disabled={!canDecide}
        onClick={onRefuse}
        sx={touchTargetSx}
      >
        {labels.refuse}
      </Button>
      <Button
        variant="contained"
        startIcon={<PublishRounded />}
        aria-label={labels.publishAccessible}
        loading={pendingDecision === 'publish'}
        disabled={!canDecide}
        onClick={onPublish}
        sx={touchTargetSx}
      >
        {labels.publish}
      </Button>
    </Stack>
  );
}
