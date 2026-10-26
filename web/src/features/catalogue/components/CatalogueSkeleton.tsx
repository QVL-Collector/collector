import { CardGrid } from '../../../ui/components/CardGrid';
import { CardGridItem } from '../../../ui/components/CardGridItem';
import { ArticleCardSkeleton } from './ArticleCardSkeleton';

const placeholderKeys = ['p1', 'p2', 'p3', 'p4', 'p5', 'p6', 'p7', 'p8'] as const;

export function CatalogueSkeleton() {
  return (
    <CardGrid hiddenFromAssistiveTechnologies>
      {placeholderKeys.map((key, index) => (
        <CardGridItem key={key} index={index}>
          <ArticleCardSkeleton />
        </CardGridItem>
      ))}
    </CardGrid>
  );
}
