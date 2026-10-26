import { useTranslation } from 'react-i18next';
import type { SubmissionFailure } from '../hooks/useSubmitForControl';

type SubmissionFailureMessageProps = {
  readonly failure: SubmissionFailure;
};

export function SubmissionFailureMessage({ failure }: SubmissionFailureMessageProps) {
  const { t } = useTranslation();
  switch (failure) {
    case 'conflict':
      return t('seller.submission.conflict');
    case 'locked':
      return t('seller.submission.locked');
    case 'unavailable':
      return t('seller.submission.failure');
  }
}
