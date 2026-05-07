import type {
  DraftItemModel,
  MessageItemModel,
  ProfileWorkModel,
  TopicCardModel,
  VideoCardModel
} from "../types/models";

export const featuredTopics: TopicCardModel[] = [
  { id: "1", title: "春日通勤穿搭", subtitle: "今日热榜第 1", heat: "326.4 万" },
  { id: "2", title: "城市夜跑记录", subtitle: "创作者都在拍", heat: "188.2 万" },
  { id: "3", title: "办公桌治愈瞬间", subtitle: "生活方式", heat: "97.8 万" }
];

export const feedVideos: VideoCardModel[] = [
  {
    id: "v1",
    authorName: "木子南",
    authorHandle: "@muzinan",
    authorAvatar: "南",
    caption: "夜色刚好，江边风也刚好，今天的城市像一首慢歌。",
    music: "原声 - 木子南",
    likes: "12.8万",
    comments: "4312",
    favorites: "2.4万",
    shares: "1890",
    likeCount: 128000,
    commentCount: 4312,
    favoriteCount: 24000,
    shareCount: 1890,
    cover:
      "linear-gradient(180deg, rgba(18,22,33,0.18) 0%, rgba(10,10,12,0.72) 100%), radial-gradient(circle at top, #4f647f 0%, #1d2430 45%, #09090b 100%)",
    tag: "城市记录",
    duration: "00:27",
    publishedAt: "2 小时前"
  },
  {
    id: "v2",
    authorName: "阿迟的厨房",
    authorHandle: "@achi",
    authorAvatar: "迟",
    caption: "下班 15 分钟快手晚餐，番茄牛肉滑蛋饭真的很适合工作日。",
    music: "轻松料理节拍",
    likes: "8.6万",
    comments: "2091",
    favorites: "1.7万",
    shares: "864",
    likeCount: 86000,
    commentCount: 2091,
    favoriteCount: 17000,
    shareCount: 864,
    cover:
      "linear-gradient(180deg, rgba(56,40,24,0.12) 0%, rgba(20,12,8,0.7) 100%), radial-gradient(circle at top, #cb8c54 0%, #6a4328 48%, #120c09 100%)",
    tag: "美食日常",
    duration: "00:41",
    publishedAt: "今天 18:24"
  },
  {
    id: "v3",
    authorName: "Yuna Studio",
    authorHandle: "@yunastudio",
    authorAvatar: "Y",
    caption: "网页版也要有沉浸感，来看看这组极简桌搭和柔光布景。",
    music: "Lo-fi Evening",
    likes: "15.1万",
    comments: "5309",
    favorites: "5.1万",
    shares: "4120",
    likeCount: 151000,
    commentCount: 5309,
    favoriteCount: 51000,
    shareCount: 4120,
    cover:
      "linear-gradient(180deg, rgba(31,44,39,0.14) 0%, rgba(12,16,14,0.72) 100%), radial-gradient(circle at top, #7ea58d 0%, #32463d 46%, #080a09 100%)",
    tag: "空间审美",
    duration: "00:32",
    publishedAt: "昨天"
  }
];

export const messageItems: MessageItemModel[] = [
  {
    id: "m1",
    title: "新粉丝通知",
    excerpt: "阿迟的厨房 关注了你",
    time: "刚刚",
    unread: true,
    type: "follow"
  },
  {
    id: "m2",
    title: "视频收到新评论",
    excerpt: "“这个镜头转场太舒服了”",
    time: "12 分钟前",
    unread: true,
    type: "comment"
  },
  {
    id: "m3",
    title: "回复提醒",
    excerpt: "Yuna Studio 回复了你的评论",
    time: "1 小时前",
    unread: false,
    type: "reply"
  },
  {
    id: "m4",
    title: "系统通知",
    excerpt: "视频已处理完成，现已可在关注流展示",
    time: "今天 09:24",
    unread: false,
    type: "system"
  }
];

export const draftItems: DraftItemModel[] = [
  {
    id: "d1",
    title: "下班夜跑随拍",
    cover:
      "linear-gradient(180deg, rgba(38,42,58,0.12) 0%, rgba(14,16,26,0.72) 100%), radial-gradient(circle at top, #7387bf 0%, #314070 48%, #090b14 100%)",
    updatedAt: "今天 17:03",
    visibility: "公开"
  },
  {
    id: "d2",
    title: "四月桌搭记录",
    cover:
      "linear-gradient(180deg, rgba(54,49,37,0.12) 0%, rgba(18,16,12,0.72) 100%), radial-gradient(circle at top, #b99b6a 0%, #5b4932 46%, #0d0a07 100%)",
    updatedAt: "昨天 21:12",
    visibility: "私密"
  }
];

export const profileWorks: ProfileWorkModel[] = [
  {
    id: "w1",
    cover:
      "linear-gradient(180deg, rgba(26,28,32,0.16) 0%, rgba(9,10,11,0.76) 100%), radial-gradient(circle at top, #8893a4 0%, #36404d 46%, #09090b 100%)",
    title: "江边夜色漫游",
    metrics: "12.8万 赞"
  },
  {
    id: "w2",
    cover:
      "linear-gradient(180deg, rgba(33,44,39,0.16) 0%, rgba(10,12,11,0.76) 100%), radial-gradient(circle at top, #8aa48f 0%, #31443a 46%, #090a09 100%)",
    title: "治愈系桌搭布景",
    metrics: "15.1万 赞"
  },
  {
    id: "w3",
    cover:
      "linear-gradient(180deg, rgba(58,40,25,0.16) 0%, rgba(14,10,8,0.76) 100%), radial-gradient(circle at top, #cb9254 0%, #6d4629 46%, #0e0908 100%)",
    title: "15 分钟快手晚餐",
    metrics: "8.6万 赞"
  }
];
