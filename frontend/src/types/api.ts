export interface ApiResponse<T> {
  code: number;
  msg: string;
  data: T;
}

export interface LoginData {
  token: string;
  expires_at: string;
  user: ProfileData;
}

export interface ProfileData {
  id: string;
  username: string;
  nickname: string;
  avatar_url: string;
  signature: string;
  gender: number;
  birthday?: string;
  status: number;
  follow_count: number;
  follower_count: number;
  total_liked_count: number;
  work_count: number;
  favorite_count: number;
  is_followed: boolean;
  is_mutual: boolean;
  created_at: string;
}

export interface RelationUserData {
  id: string;
  username: string;
  nickname: string;
  avatar_url: string;
  signature: string;
  gender: number;
  status: number;
  follow_count: number;
  follower_count: number;
  is_followed: boolean;
  is_mutual: boolean;
  created_at: string;
}

export interface RelationUserListData {
  items: RelationUserData[];
  next_cursor?: string;
}

export interface MessageData {
  id: string;
  receiver_id: string;
  sender_id: string;
  type: number;
  related_id: string;
  content: string;
  is_read: number;
  created_at: string;
}

export interface MessageListData {
  items: MessageData[];
  next_cursor?: string;
}

export interface MessageUnreadCountData {
  items: Record<string, number>;
}

export interface DraftData {
  id: string;
  object_key: string;
  source_url: string;
  cover_url: string;
  title: string;
  tag_names: string;
  allow_comment: number;
  visibility: number;
  created_at: string;
  updated_at: string;
}

export interface DraftListData {
  items: DraftData[];
}

export interface SaveDraftInput {
  draft_id?: string;
  object_key: string;
  cover_url: string;
  title: string;
  tag_names: string;
  allow_comment: number;
  visibility: number;
}

export interface UploadCredentialData {
  object_key: string;
  upload_url: string;
  upload_method: string;
  content_type: string;
  expires_at: string;
  upload_token: string;
}

export interface UploadFileData {
  object_key: string;
}

export interface PublishVideoData {
  video_id: string;
  object_key: string;
  source_url: string;
  transcode_status: number;
}

export interface FeedVideoData {
  video_id: string;
  user_id: string;
  title: string;
  object_key: string;
  source_url: string;
  cover_url: string;
  duration_ms: number;
  allow_comment: number;
  visibility: number;
  transcode_status: number;
  audit_status: number;
  transcode_fail_reason: string;
  audit_remark: string;
  play_count: number;
  like_count: number;
  comment_count: number;
  favorite_count: number;
  created_at: string;
  updated_at: string;
  author?: FeedAuthorData;
  interact?: FeedInteractData;
}

export interface FeedAuthorData {
  id: string;
  nickname: string;
  avatar_url: string;
  username: string;
}

export interface FeedInteractData {
  is_followed: boolean;
  is_liked: boolean;
  is_favorited: boolean;
}

export interface FeedVideoListData {
  items: FeedVideoData[];
  next_cursor?: string;
}

export interface VideoResourceData {
  video_id: string;
  resolution: string;
  file_url: string;
  file_size: number;
  bitrate: number;
  created_at: string;
}

export interface VideoResourceListData {
  items: VideoResourceData[];
}

export interface UserVideoListData {
  items: FeedVideoData[];
  next_cursor?: string;
}

export interface HashtagData {
  id: string;
  name: string;
  use_count: number;
  created_at: string;
}

export interface HashtagListData {
  items: HashtagData[];
}

export interface CommentData {
  id: string;
  video_id: string;
  user_id: string;
  content: string;
  parent_id: string;
  root_id: string;
  to_user_id: string;
  like_count: number;
  is_deleted: boolean;
  created_at: string;
}

export interface CommentListData {
  items: CommentData[];
  next_cursor?: string;
}

export interface SearchUserData {
  id: string;
  username: string;
  nickname: string;
  avatar_url: string;
  signature: string;
  follower_count: number;
  follow_count: number;
  work_count: number;
  is_followed: boolean;
  is_mutual: boolean;
}

export interface SearchUsersResponseData {
  items: SearchUserData[];
  next_cursor?: string;
}

export interface SearchHashtagData {
  id: string;
  name: string;
  use_count: number;
}

export interface SearchHashtagsResponseData {
  items: SearchHashtagData[];
  next_cursor?: string;
}

export interface SearchVideoData {
  video_id: string;
  user_id: string;
  title: string;
  cover_url: string;
  source_url: string;
  play_count: number;
  like_count: number;
  comment_count: number;
  favorite_count: number;
  visibility: number;
  audit_status: number;
  transcode_status: number;
  author: FeedAuthorData;
  interact: FeedInteractData;
}

export interface SearchVideosResponseData {
  items: SearchVideoData[];
  next_cursor?: string;
}

export interface SearchAllResponseData {
  users: SearchUserData[];
  hashtags: SearchHashtagData[];
  videos: SearchVideoData[];
}
