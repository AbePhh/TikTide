# Tiktide 企业级短视频平台技术架构与实施细节文档

## 一、项目定位

Tiktide 是一个以 Golang 为后端核心的短视频平台项目，目标不是一次性复刻完整 TikTok，而是围绕“可真实落地、可稳定演示、可在面试中讲深”的范围，完成一套具备企业级后端设计风格的短视频平台核心能力。

当前迭代聚焦以下功能：

- 用户注册、登录、JWT 鉴权、退出登录、个人资料维护
- 用户关注/取关、关注列表、粉丝列表
- 视频发布、草稿箱、话题绑定
- OSS 直传、异步转码、多码率资源生成
- 关注流 Feed 拉取
- 点赞、评论、评论点赞、收藏
- Redis 缓存与 Kafka 异步落库
- 消息通知未读数、通知列表、已读处理

当前版本明确不纳入交付范围：

- 推荐流算法系统
- Elasticsearch 搜索与热搜
- 直播、推流、聊天室
- Web 管理后台
- 复杂审核平台与 AI 内容审核平台

这份范围控制是为了保证项目能按时完成，并且每一项都能做到“能跑、能测、能讲”。

## 二、总体架构

系统采用前后端分离 + 微服务拆分的实现方式，后端按业务职责拆分为 1 个网关服务和 4 个核心业务服务。

### 2.1 服务划分

#### 1. Gateway 网关服务

职责：

- 统一 HTTP 入口
- JWT 鉴权与用户身份透传
- 路由分发
- 全局限流
- Trace ID 注入
- 统一错误码与响应结构

#### 2. User 用户服务

职责：

- 用户注册、登录、退出登录
- 用户资料查询与编辑
- 密码修改
- 用户统计信息维护
- 关注/取关关系维护
- 粉丝/关注列表查询

#### 3. Video 视频服务

职责：

- 获取 OSS 直传凭证
- 发布视频元数据
- 草稿箱管理
- 话题绑定
- 视频详情查询
- 视频多码率资源管理
- 发起转码任务
- 转码结果回写

#### 4. Interact 互动服务

职责：

- 视频点赞/取消点赞
- 视频收藏/取消收藏
- 评论发布与列表查询
- 评论点赞/取消点赞
- 互动事件写入 Kafka
- 消息通知入库与未读数维护

#### 5. Feed 关注流服务

职责：

- 维护关注流收件箱/发件箱
- 订阅视频发布成功事件
- 生成关注流
- 按游标分页拉取 Feed

### 2.2 架构分层

- 终端层：移动端或 Web Demo，负责登录、上传、刷关注流、互动操作
- 网关层：Gateway，对外暴露 RESTful API
- 服务层：User、Video、Interact、Feed 四个业务服务，服务间使用 gRPC 通信
- 基础设施层：MySQL、Redis、Kafka、MinIO/OSS、FFmpeg、Etcd、Jaeger

## 三、项目目录结构

采用 Monorepo 单仓模式，目录建议如下：

```plaintext
tiktide
├── app/
│   ├── gateway/
│   │   ├── etc/
│   │   ├── internal/
│   │   └── gateway.go
│   ├── user/
│   │   ├── api/
│   │   ├── rpc/
│   │   └── model/
│   ├── video/
│   │   ├── api/
│   │   ├── rpc/
│   │   ├── job/
│   │   └── model/
│   ├── interact/
│   │   ├── api/
│   │   ├── rpc/
│   │   ├── mq/
│   │   └── model/
│   └── feed/
│       ├── api/
│       ├── rpc/
│       ├── mq/
│       └── model/
├── common/
│   └── proto/
├── pkg/
│   ├── config/
│   ├── errno/
│   ├── middleware/
│   ├── trace/
│   ├── kafka/
│   ├── rediskey/
│   ├── jwt/
│   ├── oss/
│   └── utils/
├── script/
│   ├── sql/
│   ├── ffmpeg/
│   ├── docker/
│   └── deploy/
└── go.mod
```

## 四、技术栈选型

### 4.1 后端与服务治理

|技术领域|技术选型|说明|
|---|---|---|
|后端语言|Golang 1.24+|主开发语言，兼顾性能、并发与工程落地效率|
|微服务框架|go-zero|生成 API/RPC 骨架，统一中间件、限流、熔断与服务治理|
|服务通信|gRPC + Protobuf|内部服务通信标准，减少序列化与网络开销|
|服务发现|Etcd|服务注册与发现|
|链路追踪|OpenTelemetry + Jaeger|跨服务 Trace 透传与链路排障|
|日志|Zap|结构化日志输出|

### 4.2 数据与中间件

|技术领域|技术选型|说明|
|---|---|---|
|主数据库|MySQL 8.0|核心业务数据持久化|
|缓存|Redis 6.x|热点数据、互动状态、Feed inbox/outbox、未读数缓存|
|消息队列|Kafka 3.x|转码任务、Feed 分发、互动异步落库、通知事件|
|对象存储|MinIO / 阿里云 OSS|视频原文件、转码结果、封面、头像等静态资源存储|
|视频处理|FFmpeg|异步转码、封面截帧、多码率输出|

## 五、服务边界与核心数据流

### 5.1 用户与关注关系

- `t_user`、`t_user_stats`、`t_relation` 归 User 服务管理
- Feed 服务通过 User RPC 获取用户关注列表或关注关系变更
- 关注/取关成功后，User 服务负责刷新关系缓存

### 5.2 视频发布与转码

- `t_video`、`t_video_resource`、`t_draft`、`t_hashtag`、`t_video_hashtag` 归 Video 服务管理
- 客户端先申请上传凭证，再将原视频直传到 OSS
- 客户端调用发布接口提交 `object_key` 与业务元数据
- Video 服务落库后投递 `topic.video.transcode`
- 转码任务成功后回写主表与资源表，并投递 `topic.video.ready`
- Feed 服务消费 `topic.video.ready`，将可见视频写入关注流

### 5.3 点赞、评论、收藏与通知

- `t_like`、`t_favorite`、`t_comment`、`t_comment_like`、`t_message` 归 Interact 服务管理
- 请求到达时优先更新 Redis 互动状态和计数
- 互动事件写入 Kafka 后异步批量落库
- 对需要通知的事件，写入 `topic.notify`
- Interact 服务消费者负责生成通知记录和未读数缓存

### 5.4 关注流

- Feed 服务只负责“关注流”，不负责推荐流
- 普通作者采用“有限写扩散”
- 粉丝量较大的作者采用“作者发件箱 + 读时合并”
- Feed 拉取使用游标分页，减少深分页开销

## 六、Kafka Topic 规划

|Topic|生产者|消费者|用途|
|---|---|---|---|
|`topic.video.transcode`|Video|Video Job Worker|异步转码任务|
|`topic.video.ready`|Video Job Worker|Feed|视频转码成功并可进入关注流|
|`topic.interact.action`|Interact API|Interact MQ Worker|点赞、评论、收藏异步批量落库|
|`topic.notify`|User / Interact / Video|Interact MQ Worker|关注、点赞、评论、视频处理结果通知|

说明：

- `topic.interact.action` 中消息体需要包含 `action_type`、`biz_type`、`uid`、`target_id`、`op_time`
- `topic.notify` 中消息体需要包含 `receiver_id`、`sender_id`、`notify_type`、`related_id`、`content`

## 七、Redis Key 规划

|场景|Key|类型|说明|
|---|---|---|---|
|用户基础信息|`user:info:{uid}`|Hash|昵称、头像、状态等|
|用户统计|`user:stats:{uid}`|Hash|关注数、粉丝数、获赞数、作品数|
|关注集合|`following:{uid}`|Set|当前用户关注的人|
|粉丝集合|`followers:{uid}`|Set|当前用户的粉丝|
|视频统计|`video:stats:{vid}`|Hash|点赞数、评论数、收藏数、播放数|
|视频点赞用户集合|`video:liked:{vid}`|Set|视频点赞防重|
|视频收藏用户集合|`video:favorited:{vid}`|Set|视频收藏防重|
|评论点赞用户集合|`comment:liked:{cid}`|Set|评论点赞防重|
|关注流收件箱|`feed:inbox:{uid}`|ZSet|用户关注流收件箱|
|作者发件箱|`feed:outbox:{uid}`|ZSet|大 V 或活跃作者发件箱|
|消息未读数|`msg:unread:{uid}`|Hash|按类型维护未读数|
|JWT 黑名单|`jwt:blacklist:{token}`|String|退出登录后失效 Token|

## 八、关键业务链路

### 8.1 视频直传与异步转码链路

1. 客户端调用 `POST /api/v1/video/upload-credential` 获取上传地址与 `object_key`
2. 客户端直传视频原文件到 OSS
3. 客户端调用 `POST /api/v1/video/publish` 提交 `object_key`、标题、话题、权限等元数据
4. Video 服务校验对象是否存在，写入 `t_video`
5. Video 服务投递 `topic.video.transcode`
6. Worker 使用 FFmpeg 生成 480p/720p/1080p 输出与封面
7. Worker 回写 `t_video_resource`、`cover_url`、`transcode_status`
8. 若审核状态可见，则投递 `topic.video.ready`
9. Feed 服务消费成功事件，将视频分发到关注流
10. Video 服务通过通知事件告知作者“视频已处理完成”或“处理失败”

### 8.2 关注流生成链路

发布成功后：

- 普通作者：向最近活跃粉丝的 `feed:inbox:{uid}` 写入视频
- 粉丝量较大的作者：仅写入 `feed:outbox:{author_uid}`

拉取关注流时：

1. Feed 服务先读取当前用户的 `feed:inbox:{uid}`
2. 对用户关注的大 V，再并发读取其 `feed:outbox:{author_uid}`
3. 在内存中做合并、去重、按时间倒序排序
4. 批量调用 Video 与 User RPC 组装展示数据
5. 返回游标 `next_cursor`

### 8.3 点赞/评论/收藏链路

1. 请求经过 Gateway 鉴权、限流后进入 Interact 服务
2. 优先在 Redis 中校验幂等性和当前状态
3. 成功后先更新 Redis 计数与状态，快速返回客户端
4. 异步写入 `topic.interact.action`
5. MQ Worker 按批次合并操作并落库
6. 需要通知的行为再写入 `topic.notify`

### 8.4 通知链路

通知来源：

- 新增粉丝
- 视频被点赞
- 评论/回复
- 视频转码成功
- 视频转码失败

处理流程：

1. 业务服务投递 `topic.notify`
2. Interact 消费者生成 `t_message`
3. 同步递增 `msg:unread:{uid}` 对应类型计数
4. 前端拉取消息列表时查库，拉取未读数时优先查 Redis

## 九、稳定性与安全设计

### 9.1 鉴权与安全

- JWT 采用 RS256 签名
- Gateway 层统一校验 Token，有效用户信息透传到下游
- 退出登录时将 Token 放入黑名单，过期时间与 JWT 剩余寿命一致
- 所有写接口都要做参数校验、限流与幂等控制
- 密码使用 bcrypt，不记录明文密码

### 9.2 一致性设计

- Redis 作为高频读写入口，MySQL 作为最终持久化基准
- Kafka 消费失败需支持重试与死信处理
- 定时任务对视频计数、互动计数、未读数执行补偿校正

### 9.3 可观测性

- 全链路 Trace ID 透传
- 关键动作打点：登录、发布、转码开始/完成、Feed 拉取、互动成功、通知生成
- 监控项：接口耗时、Kafka 积压、Redis 命中率、MySQL 慢查询、FFmpeg 任务失败率

## 十、当前版本的简历亮点

这套架构在实习简历中应重点强调以下几件事：

- 使用 Golang + go-zero 搭建微服务风格短视频平台后端
- 通过 OSS 直传 + Kafka 异步转码 + FFmpeg 多码率输出处理大文件发布链路
- 通过 Redis + Kafka 异步批量落库实现点赞、评论、收藏等高并发写优化
- 基于推拉结合模型实现关注流 Feed
- 使用消息通知体系串联关注、互动、视频处理结果

这比“功能很多但都做得很浅”的项目更适合面试。
