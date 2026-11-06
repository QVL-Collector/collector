import type { ArticleReason } from '../../../shared/articleReason';
import type { NotificationIssue } from '../domain/sellerNotification';

export type DemoNotification = {
  readonly id: string;
  readonly articleId: string;
  readonly revision: number;
  readonly issue: NotificationIssue;
  readonly motif: ArticleReason | null;
  readonly decideLe: string;
};

export const demoNotifications: readonly DemoNotification[] = [
  {
    id: '3c8f1a2b-6d4e-4f9a-8b7c-1e2d3f4a5b01',
    articleId: 'c2e9b4a7-1d6f-4f80-9a3b-8c5e7d2f1a03',
    revision: 2,
    issue: 'publie',
    motif: null,
    decideLe: '2026-11-04T09:42:00.000000Z',
  },
  {
    id: '3c8f1a2b-6d4e-4f9a-8b7c-1e2d3f4a5b02',
    articleId: '9a3d7c1e-5b2f-4a68-8e4d-1f7b6c9a2d04',
    revision: 2,
    issue: 'a_verifier',
    motif: 'categorie_inconnue',
    decideLe: '2026-11-03T16:05:00.000000Z',
  },
  {
    id: '3c8f1a2b-6d4e-4f9a-8b7c-1e2d3f4a5b03',
    articleId: '7f4a1d2e-9c8b-4e57-b3a0-6e2f8d1c4b02',
    revision: 3,
    issue: 'refuse',
    motif: 'annonce_incomplete',
    decideLe: '2026-11-02T11:18:00.000000Z',
  },
];
