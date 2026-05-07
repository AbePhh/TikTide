# TikTide 搜索模块当前落地说明

## 本轮已实现

- 后端已新增独立搜索模块：`backend/internal/search`
- 已接入 Elasticsearch 官方 Go Client：`github.com/elastic/go-elasticsearch/v8`
- 已新增搜索配置项：
  - `SEARCH_ENABLED`
  - `SEARCH_ELASTIC_ADDRESSES`
  - `SEARCH_ELASTIC_USERNAME`
  - `SEARCH_ELASTIC_PASSWORD`
  - `SEARCH_USERS_ALIAS`
  - `SEARCH_HASHTAGS_ALIAS`
  - `SEARCH_VIDEOS_ALIAS`
  - `SEARCH_USE_IK`
  - `SEARCH_BOOTSTRAP_REBUILD`
- 已支持索引初始化：
  - `tiktide_users_v1` + alias `tiktide_users`
  - `tiktide_hashtags_v1` + alias `tiktide_hashtags`
  - `tiktide_videos_v1` + alias `tiktide_videos`
- 已实现 4 个搜索接口：
  - `GET /api/v1/search/users`
  - `GET /api/v1/search/hashtags`
  - `GET /api/v1/search/videos`
  - `GET /api/v1/search/all`
- 已实现搜索游标分页：
  - 基于 Elasticsearch `search_after`
  - 对外以字符串 `cursor` 返回
- 已实现索引文档同步钩子：
  - 用户注册成功后同步用户索引
  - 修改 username 后同步用户索引
  - 修改 profile 后同步用户索引
  - 创建话题后同步话题索引
  - 发布视频后同步视频索引
  - 视频转码成功后再次同步视频索引
- 已实现启动期全量回填：
  - 当 `SEARCH_ENABLED=true` 且 `SEARCH_BOOTSTRAP_REBUILD=true` 时
  - 服务启动会自动初始化索引并从 MySQL 回填用户、话题、视频数据

## 当前行为说明

- Elasticsearch 主要负责“召回 + 排序 + 分页”
- 搜索结果展示字段优先从 MySQL 现查补全
- 这样可以避免点赞数、播放数、昵称等字段完全依赖 ES 冗余同步
- 视频搜索已强制过滤：
  - `visibility = 1`
  - `audit_status = 1`
  - `transcode_status = 2`

## 当前还没做

- 还没有热搜词接口
- 还没有搜索建议词单独接口
- 还没有搜索历史
- 还没有高亮片段返回
- 还没有异步重试 / 死信式索引补偿
- 还没有独立命令式“索引重建工具”
- 还没有前端搜索页接入
- 还没有对互动数据做 ES 热度实时增量同步

## 建议你本地启用方式

在 `backend/.env` 中新增例如：

```env
SEARCH_ENABLED=true
SEARCH_ELASTIC_ADDRESSES=http://127.0.0.1:9200
SEARCH_ELASTIC_USERNAME=
SEARCH_ELASTIC_PASSWORD=
SEARCH_USERS_ALIAS=tiktide_users
SEARCH_HASHTAGS_ALIAS=tiktide_hashtags
SEARCH_VIDEOS_ALIAS=tiktide_videos
SEARCH_USE_IK=false
SEARCH_BOOTSTRAP_REBUILD=true
```

## 接下来最建议做的事

1. 先本地启动 Elasticsearch，确认服务启动时索引能自动创建。
2. 用已有数据验证：
   - 搜用户
   - 搜话题
   - 搜视频标题
   - 搜作者名找视频
3. 前端顶部搜索框接 `/api/v1/search/all`
4. 再做独立搜索结果页接 `/api/v1/search/users|hashtags|videos`
