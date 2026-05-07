# TikTide 搜索模块（Elasticsearch）详细实现文档

## 1. 文档目标

本文档用于指导 TikTide 后端新增基于 Elasticsearch 的搜索模块，实现以下能力：

- 搜索作者
- 搜索话题
- 搜索视频
- 顶部搜索框联想
- 综合搜索页
- 热搜词能力预留

本文档重点覆盖：

- 搜索模块总体架构
- Elasticsearch 索引设计
- 中文分词与搜索策略
- 后端模块拆分建议
- 接口设计
- 数据同步设计
- 索引重建与运维建议
- 前端接入建议

本文档默认基于当前仓库结构：

- 后端目录：`D:\code\TikTide\backend`
- 前端目录：`D:\code\TikTide\frontend`
- 文档目录：`D:\code\TikTide\文档`

---

## 2. 当前系统基础与设计前提

结合当前代码，TikTide 已具备以下可用于搜索的数据实体：

### 2.1 作者数据

来源表：

- `t_user`
- `t_user_stats`

现有关键字段：

- `id`
- `username`
- `nickname`
- `avatar_url`
- `signature`
- `status`
- `created_at`
- `follower_count`
- `follow_count`
- `work_count`

### 2.2 话题数据

来源表：

- `t_hashtag`

现有关键字段：

- `id`
- `name`
- `use_count`
- `created_at`

### 2.3 视频数据

来源表：

- `t_video`
- `t_video_hashtag`
- `t_hashtag`
- `t_user`

现有关键字段：

- `id`
- `user_id`
- `title`
- `cover_url`
- `source_url`
- `play_count`
- `like_count`
- `comment_count`
- `favorite_count`
- `visibility`
- `audit_status`
- `transcode_status`
- `created_at`

### 2.4 搜索设计前提

搜索模块设计遵循以下原则：

1. 不与 `user`、`video`、`feed` 业务服务强耦合。
2. Elasticsearch 作为独立搜索引擎，MySQL 仍然是主数据源。
3. 搜索索引中允许冗余作者名、话题名、统计字段，以换取查询性能和实现清晰度。
4. 第一阶段先完成“可用搜索”，后续再做“高级搜索体验”。
5. 搜索结果必须受业务状态过滤约束，不能把不可见、未转码、未审核的视频暴露给用户。

---

## 3. 总体架构设计

## 3.1 推荐架构

搜索模块拆分为 4 层：

1. 业务数据层  
   MySQL 中保存作者、话题、视频主数据。

2. 搜索索引层  
   Elasticsearch 中保存面向搜索优化后的文档。

3. 搜索服务层  
   后端新增 `internal/search` 模块，负责：
   - 查询 Elasticsearch
   - 封装搜索结果
   - 管理索引同步
   - 管理索引重建

4. HTTP 接口层  
   暴露作者搜索、话题搜索、视频搜索、综合搜索接口给前端。

## 3.2 模块边界建议

不建议：

- 把搜索逻辑放进 `user/service`
- 把搜索逻辑放进 `video/service`
- 把搜索逻辑塞进 `feed/service`

建议新增：

- `backend/internal/search/model`
- `backend/internal/search/service`
- `backend/internal/http/handler/search_handler.go`

应用上下文中新增：

- `SearchService`

---

## 4. Elasticsearch 索引设计

## 4.1 索引拆分建议

建议拆成 3 个索引：

- `tiktide_users`
- `tiktide_hashtags`
- `tiktide_videos`

不要第一版就做单一总索引，原因如下：

- 不同实体字段完全不同
- 不同实体排序逻辑不同
- 不同实体过滤规则不同
- 独立索引便于重建与问题定位

## 4.2 索引别名建议

不要直接将程序绑定到物理索引名，建议使用 alias：

- alias: `tiktide_users`
- real index: `tiktide_users_v1`

- alias: `tiktide_hashtags`
- real index: `tiktide_hashtags_v1`

- alias: `tiktide_videos`
- real index: `tiktide_videos_v1`

后续如 mapping 调整：

1. 新建 `v2` 索引
2. 全量重建数据
3. alias 指向 `v2`
4. 下线旧索引

---

## 5. 中文搜索与分词建议

## 5.1 分词路线建议

推荐分为两个可选方案：

### 方案 A：官方 `smartcn`

优点：

- 官方支持
- 部署简单
- 与 ES 版本兼容性稳定

缺点：

- 中文搜索效果中规中矩
- 对新词、短视频语义词支持一般

适用：

- 第一版先求稳定

### 方案 B：IK 分词插件

优点：

- 中文分词效果通常更好
- 支持扩展词典
- 更适合短视频内容平台

缺点：

- 第三方插件
- 需要严格匹配 Elasticsearch 版本
- 运维复杂度更高

适用：

- 明确追求中文搜索体验
- 可接受插件运维成本

## 5.2 推荐选择

如果当前目标是尽快稳定上线，建议：

- 第一版：`smartcn`
- 第二版体验优化：升级为 `IK`

如果你已经明确准备长期使用 ES 做内容搜索，也可以直接上 IK，但必须把部署版本控制好。

---

## 6. 索引字段设计

## 6.1 作者索引文档结构

索引别名：

- `tiktide_users`

建议文档：

```json
{
  "id": "2050954618547998720",
  "username": "lxn",
  "nickname": "老许南",
  "signature": "记录真实生活",
  "avatar_url": "",
  "status": 1,
  "follower_count": 1280,
  "follow_count": 215,
  "work_count": 42,
  "created_at": "2026-05-07T12:00:00Z"
}
```

作者索引关键搜索字段：

- `username`
- `nickname`
- `signature`

其中：

- `username` 更偏精确 / 前缀匹配
- `nickname` 更偏全文搜索
- `signature` 仅作为弱召回字段

## 6.2 话题索引文档结构

索引别名：

- `tiktide_hashtags`

建议文档：

```json
{
  "id": "6",
  "name": "北京",
  "use_count": 321,
  "created_at": "2026-05-07T12:00:00Z"
}
```

关键搜索字段：

- `name`

## 6.3 视频索引文档结构

索引别名：

- `tiktide_videos`

建议文档：

```json
{
  "id": "2051502832120500224",
  "title": "北京夜景随拍",
  "user_id": "2050954618547998720",
  "author_username": "lxn",
  "author_nickname": "老许南",
  "hashtags": ["北京", "夜景", "城市记录"],
  "cover_url": "https://...",
  "play_count": 10023,
  "like_count": 1234,
  "comment_count": 89,
  "favorite_count": 456,
  "visibility": 1,
  "audit_status": 1,
  "transcode_status": 2,
  "created_at": "2026-05-07T12:00:00Z"
}
```

关键搜索字段：

- `title`
- `author_username`
- `author_nickname`
- `hashtags`

冗余这些字段的目的：

- 视频搜索时可以支持搜标题
- 可以支持搜作者名找视频
- 可以支持搜话题找视频

---

## 7. Mapping 设计建议

## 7.1 通用原则

对可搜索字段，建议使用“双字段”模式：

- 一个 `text` 字段用于全文搜索
- 一个 `keyword` 子字段用于精确匹配与排序

对需要联想的字段，建议加：

- `search_as_you_type`

## 7.2 作者索引 mapping 建议

```json
{
  "settings": {
    "analysis": {
      "analyzer": {
        "default_cn": {
          "type": "smartcn"
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "username": {
        "type": "text",
        "analyzer": "default_cn",
        "fields": {
          "raw": { "type": "keyword" },
          "suggest": { "type": "search_as_you_type" }
        }
      },
      "nickname": {
        "type": "text",
        "analyzer": "default_cn",
        "fields": {
          "raw": { "type": "keyword" },
          "suggest": { "type": "search_as_you_type" }
        }
      },
      "signature": {
        "type": "text",
        "analyzer": "default_cn"
      },
      "avatar_url": { "type": "keyword", "index": false },
      "status": { "type": "byte" },
      "follower_count": { "type": "long" },
      "follow_count": { "type": "long" },
      "work_count": { "type": "long" },
      "created_at": { "type": "date" }
    }
  }
}
```

## 7.3 话题索引 mapping 建议

```json
{
  "settings": {
    "analysis": {
      "analyzer": {
        "default_cn": {
          "type": "smartcn"
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "name": {
        "type": "text",
        "analyzer": "default_cn",
        "fields": {
          "raw": { "type": "keyword" },
          "suggest": { "type": "search_as_you_type" }
        }
      },
      "use_count": { "type": "long" },
      "created_at": { "type": "date" }
    }
  }
}
```

## 7.4 视频索引 mapping 建议

```json
{
  "settings": {
    "analysis": {
      "analyzer": {
        "default_cn": {
          "type": "smartcn"
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "title": {
        "type": "text",
        "analyzer": "default_cn",
        "fields": {
          "raw": { "type": "keyword" },
          "suggest": { "type": "search_as_you_type" }
        }
      },
      "user_id": { "type": "keyword" },
      "author_username": {
        "type": "text",
        "analyzer": "default_cn",
        "fields": {
          "raw": { "type": "keyword" },
          "suggest": { "type": "search_as_you_type" }
        }
      },
      "author_nickname": {
        "type": "text",
        "analyzer": "default_cn",
        "fields": {
          "raw": { "type": "keyword" },
          "suggest": { "type": "search_as_you_type" }
        }
      },
      "hashtags": {
        "type": "text",
        "analyzer": "default_cn",
        "fields": {
          "raw": { "type": "keyword" },
          "suggest": { "type": "search_as_you_type" }
        }
      },
      "cover_url": { "type": "keyword", "index": false },
      "play_count": { "type": "long" },
      "like_count": { "type": "long" },
      "comment_count": { "type": "long" },
      "favorite_count": { "type": "long" },
      "visibility": { "type": "byte" },
      "audit_status": { "type": "byte" },
      "transcode_status": { "type": "byte" },
      "created_at": { "type": "date" }
    }
  }
}
```

---

## 8. 查询设计

## 8.1 顶部搜索框联想查询

适用于：

- 用户输入时的下拉联想
- 只返回少量结果
- 偏前缀匹配

建议使用：

- `multi_match`
- `type = bool_prefix`

作者联想示例：

```json
{
  "size": 5,
  "query": {
    "multi_match": {
      "query": "北",
      "type": "bool_prefix",
      "fields": [
        "username.suggest",
        "username.suggest._2gram",
        "username.suggest._3gram",
        "nickname.suggest",
        "nickname.suggest._2gram",
        "nickname.suggest._3gram"
      ]
    }
  }
}
```

话题和视频联想同理。

## 8.2 正式搜索页查询

适用于：

- 回车进入搜索页
- 需要更强相关性
- 需要业务排序
- 需要分页

建议：

- `bool`
- `should`
- `multi_match`
- `match_phrase`
- `function_score`

---

## 9. 搜索排序规则建议

## 9.1 作者搜索排序

建议优先级：

1. `username` 精确命中
2. `username` 前缀命中
3. `nickname` 前缀命中
4. `nickname` 普通命中
5. `follower_count` 高的靠前

## 9.2 话题搜索排序

建议优先级：

1. `name` 精确命中
2. `name` 前缀命中
3. `name` 普通命中
4. `use_count` 高的靠前

## 9.3 视频搜索排序

建议优先级：

1. 标题精确 / phrase 命中
2. 标题全文匹配
3. 作者名命中
4. 话题命中
5. 热度加权
6. 新鲜度加权

热度加权建议使用：

- `play_count`
- `like_count`
- `comment_count`
- `favorite_count`

新鲜度可使用：

- `created_at`

## 9.4 视频搜索过滤条件

视频搜索必须过滤：

- `visibility = 1`
- `audit_status = 1`
- `transcode_status = 2`

否则搜索结果中会出现不可播放或不可见视频。

---

## 10. 后端模块设计建议

## 10.1 新增模块结构

建议新增：

```text
backend/internal/search/
├── model/
│   ├── search.go
│   └── elastic_repository.go
├── service/
│   └── service.go
```

HTTP 层新增：

```text
backend/internal/http/handler/search_handler.go
```

## 10.2 SearchService 建议接口

建议定义：

```go
type SearchService interface {
    SearchUsers(ctx context.Context, req SearchUsersRequest) (*UserSearchResult, error)
    SearchHashtags(ctx context.Context, req SearchHashtagsRequest) (*HashtagSearchResult, error)
    SearchVideos(ctx context.Context, req SearchVideosRequest) (*VideoSearchResult, error)
    SearchAll(ctx context.Context, req SearchAllRequest) (*AllSearchResult, error)

    UpsertUserDocument(ctx context.Context, userID int64) error
    UpsertHashtagDocument(ctx context.Context, hashtagID int64) error
    UpsertVideoDocument(ctx context.Context, videoID int64) error

    DeleteUserDocument(ctx context.Context, userID int64) error
    DeleteHashtagDocument(ctx context.Context, hashtagID int64) error
    DeleteVideoDocument(ctx context.Context, videoID int64) error

    RebuildUserIndex(ctx context.Context) error
    RebuildHashtagIndex(ctx context.Context) error
    RebuildVideoIndex(ctx context.Context) error
}
```

## 10.3 Repository 层职责

Repository 层只负责：

- 调用 Elasticsearch
- 封装 query DSL
- 执行 bulk
- 创建索引与 alias

Service 层负责：

- 调 MySQL 聚合构造搜索文档
- 决定同步时机
- 处理业务过滤和 DTO 输出

---

## 11. MySQL -> Elasticsearch 文档构建建议

## 11.1 用户文档构建

构建时需要聚合：

- `t_user`
- `t_user_stats`

生成作者索引文档。

## 11.2 话题文档构建

构建时读取：

- `t_hashtag`

生成话题索引文档。

## 11.3 视频文档构建

构建时需要聚合：

- `t_video`
- `t_user`
- `t_video_hashtag`
- `t_hashtag`

目的是把以下信息写进视频索引：

- 标题
- 作者用户名
- 作者昵称
- 话题名数组
- 封面
- 统计字段
- 状态字段

---

## 12. 数据同步时机设计

## 12.1 作者索引同步时机

在以下操作成功后同步：

- 用户注册成功
- 修改 username
- 修改 nickname
- 修改 signature
- 修改 avatar_url

## 12.2 话题索引同步时机

在以下操作成功后同步：

- 创建话题
- 话题 `use_count` 变化

## 12.3 视频索引同步时机

在以下操作成功后同步：

- 视频发布成功：创建基础文档
- 转码成功：补封面 / 转码状态
- 审核通过：更新审核状态
- 标题更新：更新标题
- 话题变更：更新 `hashtags`

## 12.4 第一阶段同步策略

第一阶段建议采用：

- MySQL 写成功后，同步写 Elasticsearch

优点：

- 简单
- 易调试
- 适合当前阶段

风险：

- 写 ES 失败时需要记录错误
- 需要补重试机制

## 12.5 第二阶段升级策略

后续建议升级为：

- MySQL 写成功
- 发送领域事件
- worker 异步更新 ES

这样可以降低主链路延迟。

---

## 13. 接口设计建议

## 13.1 作者搜索

```http
GET /api/v1/search/users?q=&cursor=&limit=
```

返回：

```json
{
  "items": [
    {
      "id": "2050954618547998720",
      "username": "lxn",
      "nickname": "老许南",
      "avatar_url": "",
      "follower_count": 1280
    }
  ],
  "next_cursor": ""
}
```

## 13.2 话题搜索

```http
GET /api/v1/search/hashtags?q=&cursor=&limit=
```

返回：

```json
{
  "items": [
    {
      "id": "6",
      "name": "北京",
      "use_count": 321
    }
  ],
  "next_cursor": ""
}
```

## 13.3 视频搜索

```http
GET /api/v1/search/videos?q=&cursor=&limit=
```

返回建议复用视频卡片字段风格。

## 13.4 综合搜索

```http
GET /api/v1/search/all?q=
```

返回：

```json
{
  "users": [],
  "hashtags": [],
  "videos": []
}
```

适用：

- 顶部搜索框联想
- 搜索结果页“综合”tab

## 13.5 热搜词接口（后续）

```http
GET /api/v1/search/hot
```

说明：

- 第一阶段可不实现
- 后续用 Redis / MySQL 维护搜索热词榜

---

## 14. 分页方案建议

ES 推荐使用：

- `search_after`

不建议大量使用深分页 `from + size`。

因此建议接口中的 `cursor` 实际表示：

- 编码后的 `search_after` 值

后端可封装成字符串，例如 base64 JSON。

这样可以避免：

- 大偏移分页性能差
- 结果不稳定

---

## 15. 索引初始化与重建建议

## 15.1 初始化流程

首次上线搜索模块时建议：

1. 创建 `v1` 物理索引
2. 建立 alias
3. 全量导入作者
4. 全量导入话题
5. 全量导入视频
6. 开启在线同步

## 15.2 全量导入方式

建议使用：

- Elasticsearch Bulk API

不要单条写入全量历史数据。

## 15.3 重建流程

标准流程建议：

1. 创建 `v2` 索引
2. 用 bulk 全量重建
3. 校验数据量与样本查询
4. 切换 alias
5. 删除旧索引

---

## 16. 运维与配置建议

## 16.1 配置项建议

建议在配置中新增：

- `search.enabled`
- `search.elastic.addresses`
- `search.elastic.username`
- `search.elastic.password`
- `search.elastic.users_alias`
- `search.elastic.hashtags_alias`
- `search.elastic.videos_alias`
- `search.elastic.use_ik`

## 16.2 降级策略建议

当 Elasticsearch 不可用时：

- 顶部搜索接口返回明确错误
- 前端提示“搜索暂不可用”
- 不建议自动 fallback 到 MySQL 临时搜索

原因：

- MySQL fallback 会导致行为不一致
- 增加维护复杂度

如果后续必须降级，也应作为独立设计，不建议第一版就引入。

---

## 17. 前端接入建议

## 17.1 顶部搜索框

建议改造为：

1. 输入防抖 `300~400ms`
2. 长度小于 2 不查询
3. 调用 `/api/v1/search/all?q=`
4. 下拉展示：
   - 作者 3 条
   - 话题 3 条
   - 视频 3 条
5. 回车进入 `/search?q=xxx`

## 17.2 搜索结果页结构建议

建议新增搜索结果页，包含：

- 综合
- 作者
- 话题
- 视频

### 综合页

展示：

- 作者前 3
- 话题前 3
- 视频前 6

### 作者页

卡片展示：

- 头像
- `nickname`
- `username`
- 粉丝数

### 话题页

卡片展示：

- 话题名
- 使用次数

### 视频页

卡片展示：

- 封面
- 标题
- 作者
- 播放量 / 点赞数

---

## 18. 热搜词设计建议（预留）

热搜词不建议第一版依赖 Elasticsearch 聚合现算。

推荐方案：

- Redis 计数搜索词
- 定时聚合到 MySQL 或直接保存在 Redis ZSET
- 对外暴露 `/api/v1/search/hot`

优点：

- 实现简单
- 更新快
- 不影响 ES 查询链路

---

## 19. 安全与过滤建议

搜索模块必须遵守业务可见性规则。

### 视频结果过滤要求

必须过滤掉：

- 私密视频
- 未审核通过视频
- 转码未成功视频
- 逻辑删除视频

### 作者结果过滤要求

可过滤：

- 被封禁用户
- 逻辑删除用户

### 话题结果过滤要求

一般无需特殊过滤，但可过滤：

- 非法或下线话题（如后续引入）

---

## 20. 测试建议

## 20.1 单元测试

建议覆盖：

- query DSL 构建
- search_after cursor 编解码
- 搜索结果 DTO 映射
- MySQL -> ES 文档构建

## 20.2 集成测试

建议覆盖：

- 创建索引
- 写入样本数据
- 作者搜索
- 话题搜索
- 视频搜索
- 综合搜索
- alias 切换

## 20.3 联调测试重点

重点验证：

- 顶部搜索联想是否稳定
- 中文短词是否可搜
- 作者 / 话题 / 视频分类结果是否正确
- 搜出的视频是否都可正常播放与访问

---

## 21. 实施优先级建议

### P0

- 引入 Elasticsearch 客户端
- 新增 `internal/search` 模块
- 新建 3 个索引
- 新增：
  - `/api/v1/search/users`
  - `/api/v1/search/hashtags`
  - `/api/v1/search/videos`
  - `/api/v1/search/all`
- 完成用户 / 话题 / 视频索引同步

### P1

- 顶部搜索联想
- 搜索结果页
- 视频结果排序优化
- alias + 重建脚本

### P2

- 热搜词
- 搜索日志
- 点击反馈
- 拼音搜索
- 错别字容忍
- IK 分词升级

---

## 22. 最终建议

对 TikTide 当前阶段，最合理的 Elasticsearch 搜索落地方案是：

1. 使用 3 个独立索引：
   - 用户
   - 话题
   - 视频
2. 使用 alias 管理索引版本
3. 第一版先用 `smartcn` 稳定上线
4. 顶部搜索框使用联想查询
5. 正式搜索页使用独立接口与业务排序
6. 搜索模块独立成 `internal/search`
7. 第一版先做同步写 ES，后续再升级异步化

这样可以在不破坏你当前业务结构的前提下，把搜索能力整洁地接入现有系统，并保留后续扩展空间。

---

## 23. 后续可继续产出的配套文档

如果继续推进，下一步建议补充以下文档：

1. 《TikTide 搜索模块（Elasticsearch）索引 Mapping 与 Alias 初始化脚本》
2. 《TikTide 搜索接口定义与返回 DTO 详细文档》
3. 《TikTide 搜索模块后端代码实现步骤说明》
4. 《TikTide 前端搜索页与联想框接入说明》

