import Stack from '@mui/material/Stack';
import { LiveStatus } from '../../../ui/components/LiveStatus';
import { PageHeading } from '../../../ui/components/PageHeading';
import { useOrderConfirmationPage } from '../hooks/useOrderConfirmationPage';
import { OrderConfirmationContent } from './OrderConfirmationContent';

export function OrderConfirmationPage() {
  const { headingRef, title, subtitle, announcement, ...content } = useOrderConfirmationPage();
  return (
    <Stack spacing={{ xs: 3, md: 5 }}>
      <PageHeading headingRef={headingRef} title={title} subtitle={subtitle} />
      <LiveStatus message={announcement} />
      <Stack alignItems={content.view.status === 'not-found' ? 'flex-start' : 'stretch'}>
        <OrderConfirmationContent {...content} />
      </Stack>
    </Stack>
  );
}
