import type { FeedTab } from "../types/models";

interface FeedTabsProps {
  activeTab: FeedTab;
  onChange: (tab: FeedTab) => void;
}

const tabs: FeedTab[] = ["推荐", "关注", "精选"];

export function FeedTabs({ activeTab, onChange }: FeedTabsProps) {
  return (
    <div className="feed-tabs">
      {tabs.map((tab) => (
        <button
          key={tab}
          type="button"
          className={tab === activeTab ? "feed-tab feed-tab-active" : "feed-tab"}
          onClick={() => onChange(tab)}
        >
          {tab}
        </button>
      ))}
    </div>
  );
}
