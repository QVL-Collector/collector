import { useEffect, useRef, useState } from 'react';
import type { ArticleState, SellerArticle } from '../domain/sellerArticle';
import { useArticleLabels } from './useArticleLabels';

type KnownStates = ReadonlyMap<string, ArticleState>;

function statesOf(articles: readonly SellerArticle[]): KnownStates {
  return new Map(articles.map((article): [string, ArticleState] => [article.id, article.state]));
}

function changedArticles(
  articles: readonly SellerArticle[],
  previous: KnownStates,
): readonly SellerArticle[] {
  return articles.filter((article) => {
    const previousState = previous.get(article.id);
    return previousState !== undefined && previousState !== article.state;
  });
}

export function useStateChangeAnnouncement(
  articles: readonly SellerArticle[] | undefined,
): string {
  const { stateSentence } = useArticleLabels();
  const knownStates = useRef<KnownStates | null>(null);
  const [announcement, setAnnouncement] = useState('');
  useEffect(() => {
    if (articles === undefined) {
      return;
    }
    const previous = knownStates.current;
    knownStates.current = statesOf(articles);
    const changed = previous === null ? [] : changedArticles(articles, previous);
    if (changed.length > 0) {
      setAnnouncement(changed.map(stateSentence).join(' '));
    }
  }, [articles, stateSentence]);
  return announcement;
}
