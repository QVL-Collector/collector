import ManageSearchRounded from '@mui/icons-material/ManageSearchRounded';
import Card from '@mui/material/Card';
import Chip from '@mui/material/Chip';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { CardTitle } from '../../../ui/components/CardTitle';
import { FactItem } from '../../../ui/components/FactItem';
import { FactList } from '../../../ui/components/FactList';
import type { ReviewArticle } from '../domain/reviewArticle';
import { useReviewArticleCard } from '../hooks/useReviewArticleCard';
import type { DecisionListener } from '../hooks/useReviewDecision';
import { DecisionFailureMessage } from './DecisionFailureMessage';
import { RefusalDialog } from './RefusalDialog';
import { ReviewDecisionActions } from './ReviewDecisionActions';

type ReviewArticleCardProps = {
  readonly article: ReviewArticle;
  readonly onDecided: DecisionListener;
};

export function ReviewArticleCard({ article, onDecided }: ReviewArticleCardProps) {
  const { labels, facts, decision, refusal } = useReviewArticleCard(article, onDecided);
  return (
    <Card component="article" variant="outlined">
      <Stack spacing={2.5} padding={{ xs: 2, md: 3 }}>
        <Stack
          direction={{ xs: 'column', sm: 'row' }}
          spacing={1}
          justifyContent="space-between"
          alignItems={{ xs: 'flex-start', sm: 'center' }}
        >
          <Chip
            icon={<ManageSearchRounded />}
            label={labels.controlReason}
            color="secondary"
            variant="outlined"
          />
          <Typography variant="body2" color="text.secondary">
            {labels.revision}
          </Typography>
        </Stack>
        <CardTitle level="h3">{labels.title}</CardTitle>
        <FactList>
          {facts.map(({ label, value }) => (
            <FactItem key={label} label={label}>
              {value}
            </FactItem>
          ))}
        </FactList>
        <DecisionFailureMessage failure={decision.failure} onReload={decision.reload} />
        <ReviewDecisionActions
          labels={labels}
          pendingDecision={decision.pendingDecision}
          canDecide={decision.canDecide}
          onPublish={decision.publish}
          onRefuse={refusal.openDialog}
        />
      </Stack>
      <RefusalDialog title={labels.title} model={refusal} />
    </Card>
  );
}
