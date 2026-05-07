import type {
  ApiResponse,
  CommentData,
  CommentListData,
  DraftData,
  DraftListData,
  FeedVideoListData,
  HashtagData,
  HashtagListData,
  LoginData,
  MessageListData,
  MessageUnreadCountData,
  ProfileData,
  PublishVideoData,
  RelationUserListData,
  SaveDraftInput,
  SearchAllResponseData,
  SearchHashtagsResponseData,
  SearchUsersResponseData,
  SearchVideosResponseData,
  UserVideoListData,
  UploadCredentialData,
  VideoResourceListData
} from "../types/api";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://127.0.0.1:8080";
const TOKEN_KEY = "tiktide_token";

export interface ApiGap {
  feature: string;
  reason: string;
  suggestedEndpoint?: string;
}

export const apiGaps: ApiGap[] = [];

export class ApiError extends Error {
  code: number;
  status: number;

  constructor(message: string, code: number, status: number) {
    super(message);
    this.name = "ApiError";
    this.code = code;
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = window.localStorage.getItem(TOKEN_KEY);
  const headers = new Headers(init?.headers);
  headers.set("Content-Type", "application/json");
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  const url = `${API_BASE_URL}${path}`;
  console.info("[api] request", {
    path,
    url,
    method: init?.method ?? "GET",
    hasToken: Boolean(token),
    tokenPreview: token ? `${token.slice(0, 16)}...` : null
  });

  const response = await fetch(url, {
    ...init,
    headers
  });

  const rawText = await response.text();
  let payload: ApiResponse<T>;
  try {
    payload = JSON.parse(rawText) as ApiResponse<T>;
  } catch (error) {
    console.error("[api] invalid json response", {
      path,
      url,
      method: init?.method ?? "GET",
      status: response.status,
      rawText,
      error
    });
    throw new ApiError(`响应解析失败: ${response.status}`, response.status, response.status);
  }

  if (!response.ok || payload.code !== 0) {
    console.error("[api] response error", {
      path,
      url,
      method: init?.method ?? "GET",
      status: response.status,
      code: payload.code,
      msg: payload.msg,
      hasToken: Boolean(token),
      rawText
    });
    throw new ApiError(payload.msg ?? "请求失败", payload.code ?? response.status, response.status);
  }

  return payload.data;
}

export async function login(username: string, password: string) {
  return request<LoginData>("/api/v1/user/login", {
    method: "POST",
    body: JSON.stringify({ username, password })
  });
}

export async function register(username: string, password: string) {
  return request<{ user: ProfileData }>("/api/v1/user/register", {
    method: "POST",
    body: JSON.stringify({ username, password })
  });
}

export async function getProfile() {
  return request<ProfileData>("/api/v1/user/profile");
}

export async function updateUsername(username: string) {
  return request<ProfileData>("/api/v1/user/username", {
    method: "PUT",
    body: JSON.stringify({ username })
  });
}

export async function updateProfile(input: { nickname?: string; avatar_url?: string; signature?: string }) {
  return request<ProfileData>("/api/v1/user/profile", {
    method: "PUT",
    body: JSON.stringify(input)
  });
}

export async function changePassword(input: { old_password: string; new_password: string }) {
  return request<{ changed: boolean }>("/api/v1/user/password", {
    method: "PUT",
    body: JSON.stringify(input)
  });
}

export async function logout() {
  return request<{ logged_out: boolean }>("/api/v1/user/logout", {
    method: "POST"
  });
}

export async function listFollowingUsers(userId: string, params?: { cursor?: string; limit?: number }) {
  const search = new URLSearchParams();
  if (params?.cursor) {
    search.set("cursor", params.cursor);
  }
  if (typeof params?.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<RelationUserListData>(`/api/v1/relation/following/${userId}${search.size ? `?${search.toString()}` : ""}`);
}

export async function listFollowerUsers(userId: string, params?: { cursor?: string; limit?: number }) {
  const search = new URLSearchParams();
  if (params?.cursor) {
    search.set("cursor", params.cursor);
  }
  if (typeof params?.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<RelationUserListData>(`/api/v1/relation/followers/${userId}${search.size ? `?${search.toString()}` : ""}`);
}

export async function listUserVideos(userId: string, params?: { cursor?: string; limit?: number }) {
  const search = new URLSearchParams();
  if (params?.cursor) {
    search.set("cursor", params.cursor);
  }
  if (typeof params?.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<UserVideoListData>(`/api/v1/user/${userId}/videos${search.size ? `?${search.toString()}` : ""}`);
}

export async function listMessages(params?: { type?: number; cursor?: string; limit?: number }) {
  const search = new URLSearchParams();
  if (typeof params?.type === "number") {
    search.set("type", String(params.type));
  }
  if (params?.cursor) {
    search.set("cursor", params.cursor);
  }
  if (typeof params?.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<MessageListData>(`/api/v1/message/list${search.size ? `?${search.toString()}` : ""}`);
}

export async function getUnreadCount() {
  return request<MessageUnreadCountData>("/api/v1/message/unread-count");
}

export async function markMessageRead(input: { msg_id?: string; type?: number }) {
  return request<{ read: boolean }>("/api/v1/message/read", {
    method: "POST",
    body: JSON.stringify({
      msg_id: input.msg_id,
      type: input.type
    })
  });
}

export async function listDrafts() {
  return request<DraftListData>("/api/v1/draft/list");
}

export async function getDraft(id: string) {
  return request<DraftData>(`/api/v1/draft/${id}`);
}

export async function deleteDraft(id: string) {
  return request<{ deleted: boolean }>(`/api/v1/draft/${id}`, {
    method: "DELETE"
  });
}

export async function createUploadCredential(fileName: string, contentType: string, objectKey?: string) {
  return request<UploadCredentialData>("/api/v1/video/upload-credential", {
    method: "POST",
    body: JSON.stringify({ file_name: fileName, content_type: contentType, object_key: objectKey })
  });
}

export async function saveDraft(input: SaveDraftInput) {
  return request<DraftData>("/api/v1/draft", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function publishVideo(input: {
  object_key: string;
  title: string;
  hashtag_ids: number[];
  hashtag_names: string[];
  allow_comment: number;
  visibility: number;
}) {
  return request<PublishVideoData>("/api/v1/video/publish", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function listFollowingFeed(params?: { cursor?: string; limit?: number }) {
  const search = new URLSearchParams();
  if (params?.cursor) {
    search.set("cursor", params.cursor);
  }
  if (typeof params?.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<FeedVideoListData>(`/api/v1/feed/following${search.size ? `?${search.toString()}` : ""}`);
}

export async function listRecommendFeed(params?: { cursor?: string; limit?: number }) {
  const search = new URLSearchParams();
  if (params?.cursor) {
    search.set("cursor", params.cursor);
  }
  if (typeof params?.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<FeedVideoListData>(`/api/v1/feed/recommend${search.size ? `?${search.toString()}` : ""}`);
}

export async function listHotHashtags(params?: { limit?: number }) {
  const search = new URLSearchParams();
  if (typeof params?.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<HashtagListData>(`/api/v1/hashtag/hot${search.size ? `?${search.toString()}` : ""}`);
}

export async function getHashtag(hashtagId: string) {
  return request<HashtagData>(`/api/v1/hashtag/${hashtagId}`);
}

export async function listHashtagVideos(hashtagId: string, params?: { cursor?: string; limit?: number }) {
  const search = new URLSearchParams();
  if (params?.cursor) {
    search.set("cursor", params.cursor);
  }
  if (typeof params?.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<FeedVideoListData>(`/api/v1/hashtag/${hashtagId}/videos${search.size ? `?${search.toString()}` : ""}`);
}

export async function getUserHomepage(userId: string) {
  return request<ProfileData>(`/api/v1/user/${userId}`);
}

export async function getVideoResources(videoId: string) {
  return request<VideoResourceListData>(`/api/v1/video/${videoId}/resources`);
}

export async function reportVideoPlay(videoId: string) {
  return request<{ reported: boolean }>("/api/v1/video/play/report", {
    method: "POST",
    body: JSON.stringify({
      video_id: videoId
    })
  });
}

export async function likeVideo(videoId: string, actionType: 1 | 2) {
  return request<{ done: boolean }>("/api/v1/interact/like", {
    method: "POST",
    body: JSON.stringify({
      video_id: videoId,
      action_type: actionType
    })
  });
}

export async function favoriteVideo(videoId: string, actionType: 1 | 2) {
  return request<{ done: boolean }>("/api/v1/interact/favorite", {
    method: "POST",
    body: JSON.stringify({
      video_id: videoId,
      action_type: actionType
    })
  });
}

export async function listComments(params: { video_id: string; root_id?: string; cursor?: string; limit?: number }) {
  const search = new URLSearchParams();
  search.set("video_id", params.video_id);
  search.set("root_id", params.root_id ?? "0");
  if (params.cursor) {
    search.set("cursor", params.cursor);
  }
  if (typeof params.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<CommentListData>(`/api/v1/interact/comment/list?${search.toString()}`);
}

export async function searchAll(params: { q: string; limit?: number }) {
  const search = new URLSearchParams();
  search.set("q", params.q);
  if (typeof params.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<SearchAllResponseData>(`/api/v1/search/all?${search.toString()}`);
}

export async function searchUsers(params: { q: string; cursor?: string; limit?: number }) {
  const search = new URLSearchParams();
  search.set("q", params.q);
  if (params.cursor) {
    search.set("cursor", params.cursor);
  }
  if (typeof params.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<SearchUsersResponseData>(`/api/v1/search/users?${search.toString()}`);
}

export async function searchHashtags(params: { q: string; cursor?: string; limit?: number }) {
  const search = new URLSearchParams();
  search.set("q", params.q);
  if (params.cursor) {
    search.set("cursor", params.cursor);
  }
  if (typeof params.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<SearchHashtagsResponseData>(`/api/v1/search/hashtags?${search.toString()}`);
}

export async function searchVideos(params: { q: string; cursor?: string; limit?: number }) {
  const search = new URLSearchParams();
  search.set("q", params.q);
  if (params.cursor) {
    search.set("cursor", params.cursor);
  }
  if (typeof params.limit === "number") {
    search.set("limit", String(params.limit));
  }
  return request<SearchVideosResponseData>(`/api/v1/search/videos?${search.toString()}`);
}

export async function publishComment(input: {
  video_id: string;
  content: string;
  parent_id?: string;
  root_id?: string;
  to_user_id?: string;
}) {
  return request<CommentData>("/api/v1/interact/comment/publish", {
    method: "POST",
    body: JSON.stringify({
      video_id: input.video_id,
      content: input.content,
      parent_id: input.parent_id ?? "0",
      root_id: input.root_id ?? "0",
      to_user_id: input.to_user_id ?? "0"
    })
  });
}

export async function likeComment(commentId: string, actionType: 1 | 2) {
  return request<{ done: boolean }>("/api/v1/interact/comment/like", {
    method: "POST",
    body: JSON.stringify({
      comment_id: commentId,
      action_type: actionType
    })
  });
}

export async function fakeRequest<T>(payload: T, delay = 180): Promise<T> {
  await new Promise((resolve) => window.setTimeout(resolve, delay));
  return payload;
}
