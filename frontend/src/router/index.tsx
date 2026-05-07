import { createBrowserRouter } from "react-router-dom";

import { AppLayout } from "../layouts/AppLayout";
import { FeedPage } from "../pages/FeedPage";
import { DiscoverPage } from "../pages/DiscoverPage";
import { FollowingPage } from "../pages/FollowingPage";
import { MessagesPage } from "../pages/MessagesPage";
import { ProfilePage } from "../pages/ProfilePage";
import { DraftsPage } from "../pages/DraftsPage";
import { UploadPage } from "../pages/UploadPage";
import { HashtagDetailPage } from "../pages/HashtagDetailPage";
import { UserHomepagePage } from "../pages/UserHomepagePage";
import { SearchPage } from "../pages/SearchPage";

export const router = createBrowserRouter([
  {
    path: "/",
    element: <AppLayout />,
    children: [
      { index: true, element: <FeedPage /> },
      { path: "discover", element: <DiscoverPage /> },
      { path: "search", element: <SearchPage /> },
      { path: "discover/topics/:hid", element: <HashtagDetailPage /> },
      { path: "following", element: <FollowingPage /> },
      { path: "messages", element: <MessagesPage /> },
      { path: "profile", element: <ProfilePage /> },
      { path: "users/:uid", element: <UserHomepagePage /> },
      { path: "drafts", element: <DraftsPage /> },
      { path: "upload", element: <UploadPage /> }
    ]
  }
]);
