import type { RouteObject } from 'react-router';
import { signInCallbackPath } from '../auth/authPaths';
import { adminReviewPath } from '../features/admin/adminPaths';
import { AdminReviewPage } from '../features/admin/components/AdminReviewPage';
import { publishedArticleRoutePattern } from '../features/catalogue/cataloguePaths';
import { CataloguePage } from '../features/catalogue/components/CataloguePage';
import { NotificationsPage } from '../features/notifications/components/NotificationsPage';
import { notificationsPath } from '../features/notifications/notificationsPaths';
import { MyOrdersPage } from '../features/orders/components/MyOrdersPage';
import { OrderConfirmationPage } from '../features/orders/components/OrderConfirmationPage';
import { buyerSpacePath, orderRoutePattern } from '../features/orders/ordersPaths';
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
import { ReservableArticlePage } from './pages/ReservableArticlePage';
import { RouteErrorPage } from './pages/RouteErrorPage';
import { SignInCallbackPage } from './pages/SignInCallbackPage';
import { RequireAuthentication } from './routing/RequireAuthentication';

export const appRoutes: RouteObject[] = [
  {
    element: <AppLayout />,
    errorElement: <RouteErrorPage />,
    children: [
      { index: true, element: <CataloguePage /> },
      { path: publishedArticleRoutePattern, element: <ReservableArticlePage /> },
      { path: signInCallbackPath, element: <SignInCallbackPage /> },
      {
        element: <RequireAuthentication requiredRole="vendeur" />,
        children: [
          { path: sellerSpacePath, element: <SellerSpacePage /> },
          { path: newArticlePath, element: <NewArticlePage /> },
          { path: articleRoutePattern, element: <ArticleEditorPage /> },
          { path: notificationsPath, element: <NotificationsPage /> },
        ],
      },
      {
        element: <RequireAuthentication requiredRole="acheteur" />,
        children: [
          { path: buyerSpacePath, element: <MyOrdersPage /> },
          { path: orderRoutePattern, element: <OrderConfirmationPage /> },
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
