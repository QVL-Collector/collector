import { PageHeading } from '../../../ui/components/PageHeading';
import { useSellerSpacePage } from '../hooks/useSellerSpacePage';

export function SellerSpacePage() {
  const { headingRef, title, welcome } = useSellerSpacePage();
  return <PageHeading headingRef={headingRef} title={title} subtitle={welcome} />;
}
