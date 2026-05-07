export const mainNavItems = [
  { label: "推荐", to: "/" },
  { label: "发现", to: "/discover" },
  { label: "关注", to: "/following" },
  { label: "消息", to: "/messages" }
] as const;

export const creatorNavItems = [
  { label: "上传视频", to: "/upload" },
  { label: "草稿箱", to: "/drafts" },
  { label: "我的主页", to: "/profile" }
] as const;
