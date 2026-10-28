import type { RouteObject } from 'react-router';
import { signInCallbackPath } from '../auth/authPaths';
import { adminReviewPath } from '../features/admin/adminPaths';
import { AdminReviewPage } from '../features/admin/components/AdminReviewPage';
import { publishedArticleRoutePattern } from '../features/catalogue/cataloguePaths';
import { ArticlePage } from '../features/catalogue/components/ArticlePage';
import { CataloguePage } from '../features/catalogue/components/CataloguePage';
import { ArticleEditorPage } from '../features/seller/components/ArticleEditorPage';
import { NewArticlePage } from '../features/seller/components/NewArticlePage';
import { SellerSpacePage } from '../features/seller/components/SellerSpacePage';
import {
  articleRoutePattern,
  newArticlePath,
  sellerSpacePath,
} from '../features/seller/sellerPaths';
import { AppLayout } from './layout/AppLayout';
import { NotFoundPage } from './pages/NotFoundPage';
import { RouteErrorPage } from './pages/RouteErrorPage';
import { SignInCallbackPage } from './pages/SignInCallbackPage';
import { RequireAuthentication } from './routing/RequireAuthentication';

export const appRoutes: RouteObject[] = [
  {
    element: <AppLayout />,
    errorElement: <RouteErrorPage />,
    children: [
      { index: true, element: <CataloguePage /> },
      { path: publishedArticleRoutePattern, element: <ArticlePage /> },
      { path: signInCallbackPath, element: <SignInCallbackPage /> },
      {
        element: <RequireAuthentication requiredRole="vendeur" />,
        children: [
          { path: sellerSpacePath, element: <SellerSpacePage /> },
          { path: newArticlePath, element: <NewArticlePage /> },
          { path: articleRoutePattern, element: <ArticleEditorPage /> },
        ],
      },
      {
        element: <RequireAuthentication requiredRole="admin" />,
        children: [{ path: adminReviewPath, element: <AdminReviewPage /> }],
      },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
];
