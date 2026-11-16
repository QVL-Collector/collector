import type { ReactNode } from 'react';
import type { PublishedArticle } from '../domain/catalogue';

export type PurchaseActionRenderer = (article: PublishedArticle) => ReactNode;
