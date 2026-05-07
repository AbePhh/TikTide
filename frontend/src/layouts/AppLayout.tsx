import { Outlet, useLocation } from "react-router-dom";

import { RightRail } from "../components/RightRail";
import { Sidebar } from "../components/Sidebar";
import { Topbar } from "../components/Topbar";

const immersivePaths = new Set(["/", "/following"]);
const noRightRailPaths = new Set(["/upload", "/drafts", "/profile", "/search", "/messages"]);

export function AppLayout() {
  const location = useLocation();
  const isImmersiveFeed = immersivePaths.has(location.pathname);
  const isSearchPage = location.pathname === "/search";
  const hideRightRail =
    isImmersiveFeed || noRightRailPaths.has(location.pathname) || location.pathname === "/discover" || location.pathname.startsWith("/discover/");

  return (
    <div className={`app-shell ${isImmersiveFeed ? "app-shell-immersive" : ""}`}>
      <Sidebar />
      <div className="app-main">
        <Topbar />
        <div className={`app-content ${isImmersiveFeed ? "app-content-immersive" : ""} ${hideRightRail ? "app-content-single" : ""}`}>
          <main className={`page-stage ${isImmersiveFeed ? "page-stage-immersive" : ""} ${isSearchPage ? "page-stage-search" : ""}`}>
            <Outlet />
          </main>
          {!hideRightRail ? <RightRail /> : null}
        </div>
      </div>
    </div>
  );
}
