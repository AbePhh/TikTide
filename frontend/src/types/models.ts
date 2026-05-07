export type FeedTab = "推荐" | "关注" | "精选";

export interface VideoCardModel {
  id: string;
  authorId?: string;
  authorName: string;
  authorHandle: string;
  authorAvatar: string;
  caption: string;
  music: string;
  likes: string;
  comments: string;
  favorites: string;
  shares: string;
  likeCount: number;
  commentCount: number;
  favoriteCount: number;
  shareCount: number;
  cover: string;
  coverUrl?: string;
  tag: string;
  duration: string;
  publishedAt: string;
  isFollowed?: boolean;
  isLiked?: boolean;
  isFavorited?: boolean;
  sourceUrl?: string;
  allowComment?: boolean;
}

export interface TopicCardModel {
  id: string;
  title: string;
  subtitle: string;
  heat: string;
}

export interface MessageItemModel {
  id: string;
  title: string;
  excerpt: string;
  time: string;
  unread: boolean;
  type: "like" | "comment" | "reply" | "follow" | "system";
}

export interface DraftItemModel {
  id: string;
  title: string;
  cover: string;
  updatedAt: string;
  visibility: "公开" | "私密";
}

export interface ProfileWorkModel {
  id: string;
  cover: string;
  title: string;
  metrics: string;
}
