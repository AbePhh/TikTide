# Tiktide 企业级数据库结构设计及核心接口文档

## 一、设计原则

当前版本数据库设计只服务于以下目标能力：

- 用户与鉴权
- 关注关系
- 视频发布与草稿箱
- OSS 直传与异步转码
- 话题绑定
- 点赞、评论、收藏
- 关注流
- 消息通知

设计原则如下：

- 核心业务使用 MySQL 持久化，Redis 承担高频读写与状态缓存
- 当前后端以 Gin 单体模块化架构实现，数据库访问统一使用 GORM
- 视频原文件通过阿里云 OSS 直传，数据库只保存对象 Key 与源文件地址
- 高频统计字段适度冗余，减少聚合查询压力
- 所有核心实体保留 `created_at`、`updated_at`，必要时保留 `deleted_at`
- 表结构优先支持“当前要做的功能”，不为未实现能力预留过多复杂字段
- 视频可见性由 `transcode_status`、`audit_status`、`visibility` 三个字段共同决定

## 二、数据库表结构设计

### 2.1 用户与统计体系

#### 表：`t_user`

```sql
CREATE TABLE `t_user` (
  `id` BIGINT NOT NULL COMMENT '用户ID(雪花算法)',
  `username` VARCHAR(64) NOT NULL COMMENT '登录用户名',
  `password_hash` VARCHAR(255) NOT NULL COMMENT 'bcrypt密码摘要',
  `nickname` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '昵称',
  `avatar_url` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '头像地址',
  `signature` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '个人简介',
  `gender` TINYINT NOT NULL DEFAULT 0 COMMENT '性别:0未知,1男,2女',
  `birthday` DATE DEFAULT NULL COMMENT '生日',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '用户状态:0封禁,1正常',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户基础信息表';
```

#### 表：`t_user_stats`

```sql
CREATE TABLE `t_user_stats` (
  `id` BIGINT NOT NULL COMMENT '用户ID',
  `follow_count` BIGINT NOT NULL DEFAULT 0 COMMENT '关注数',
  `follower_count` BIGINT NOT NULL DEFAULT 0 COMMENT '粉丝数',
  `total_liked_count` BIGINT NOT NULL DEFAULT 0 COMMENT '总获赞数',
  `work_count` BIGINT NOT NULL DEFAULT 0 COMMENT '作品数',
  `favorite_count` BIGINT NOT NULL DEFAULT 0 COMMENT '被收藏数',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户统计表';
```

### 2.2 关注关系体系

#### 表：`t_relation`

```sql
CREATE TABLE `t_relation` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '关注者ID',
  `follow_id` BIGINT NOT NULL COMMENT '被关注者ID',
  `is_mutual` TINYINT NOT NULL DEFAULT 0 COMMENT '是否互关:0否,1是',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_follow` (`user_id`, `follow_id`),
  KEY `idx_follow_id` (`follow_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='关注关系表';
```

### 2.3 视频发布与转码体系

#### 表：`t_video`

```sql
CREATE TABLE `t_video` (
  `id` BIGINT NOT NULL COMMENT '视频ID(雪花算法)',
  `user_id` BIGINT NOT NULL COMMENT '作者ID',
  `object_key` VARCHAR(512) NOT NULL COMMENT '原始视频对象存储Key',
  `source_url` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '原始视频地址',
  `title` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '视频标题',
  `cover_url` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '封面地址',
  `duration_ms` INT NOT NULL DEFAULT 0 COMMENT '视频时长(毫秒)',
  `allow_comment` TINYINT NOT NULL DEFAULT 1 COMMENT '是否允许评论:0否,1是',
  `visibility` TINYINT NOT NULL DEFAULT 1 COMMENT '可见性:0仅自己,1公开',
  `transcode_status` TINYINT NOT NULL DEFAULT 0 COMMENT '转码状态:0待处理,1处理中,2成功,3失败',
  `audit_status` TINYINT NOT NULL DEFAULT 1 COMMENT '审核状态:0待审,1通过,2驳回',
  `transcode_fail_reason` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '转码失败原因',
  `audit_remark` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '审核备注',
  `play_count` BIGINT NOT NULL DEFAULT 0 COMMENT '播放量',
  `like_count` BIGINT NOT NULL DEFAULT 0 COMMENT '点赞数',
  `comment_count` BIGINT NOT NULL DEFAULT 0 COMMENT '评论数',
  `favorite_count` BIGINT NOT NULL DEFAULT 0 COMMENT '收藏数',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_object_key` (`object_key`),
  KEY `idx_user_created` (`user_id`, `created_at` DESC),
  KEY `idx_visible_feed` (`visibility`, `audit_status`, `transcode_status`, `created_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频主表';
```

设计说明：

- 发布接口提交的是 `object_key`，而不是最终转码后的 `video_url`
- 可播放视频定义为：`visibility=1 AND audit_status=1 AND transcode_status=2`
- 互动计数字段以 MySQL 为最终基准，Redis 为高频缓存

#### 表：`t_video_resource`

```sql
CREATE TABLE `t_video_resource` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `video_id` BIGINT NOT NULL COMMENT '视频ID',
  `resolution` VARCHAR(16) NOT NULL COMMENT '分辨率:1080p,720p,480p',
  `file_url` VARCHAR(512) NOT NULL COMMENT '转码后文件URL',
  `file_size` BIGINT NOT NULL DEFAULT 0 COMMENT '文件大小(字节)',
  `bitrate` INT NOT NULL DEFAULT 0 COMMENT '码率',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_video_resolution` (`video_id`, `resolution`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频多码率资源表';
```

#### 表：`t_draft`

```sql
CREATE TABLE `t_draft` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `object_key` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '草稿视频对象Key',
  `cover_url` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '草稿封面地址',
  `title` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '标题草稿',
  `tag_names` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '话题快照,逗号分隔',
  `allow_comment` TINYINT NOT NULL DEFAULT 1 COMMENT '是否允许评论',
  `visibility` TINYINT NOT NULL DEFAULT 1 COMMENT '可见性',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_updated` (`user_id`, `updated_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='草稿箱表';
```

### 2.4 话题体系

#### 表：`t_hashtag`

```sql
CREATE TABLE `t_hashtag` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(64) NOT NULL COMMENT '话题名称(不含#)',
  `use_count` BIGINT NOT NULL DEFAULT 0 COMMENT '被引用次数',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='话题表';
```

#### 表：`t_video_hashtag`

```sql
CREATE TABLE `t_video_hashtag` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `video_id` BIGINT NOT NULL COMMENT '视频ID',
  `hashtag_id` BIGINT NOT NULL COMMENT '话题ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_video_hashtag` (`video_id`, `hashtag_id`),
  KEY `idx_hashtag_id` (`hashtag_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频话题关联表';
```

### 2.5 点赞、收藏与评论体系

#### 表：`t_like`

```sql
CREATE TABLE `t_like` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `video_id` BIGINT NOT NULL COMMENT '视频ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_video` (`user_id`, `video_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频点赞表';
```

#### 表：`t_favorite`

```sql
CREATE TABLE `t_favorite` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `video_id` BIGINT NOT NULL COMMENT '视频ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_video` (`user_id`, `video_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频收藏表';
```

#### 表：`t_comment`

```sql
CREATE TABLE `t_comment` (
  `id` BIGINT NOT NULL COMMENT '评论ID(雪花算法)',
  `video_id` BIGINT NOT NULL COMMENT '视频ID',
  `user_id` BIGINT NOT NULL COMMENT '评论者ID',
  `content` TEXT NOT NULL COMMENT '评论内容',
  `parent_id` BIGINT NOT NULL DEFAULT 0 COMMENT '父评论ID,0表示顶级评论',
  `root_id` BIGINT NOT NULL DEFAULT 0 COMMENT '根评论ID,0表示顶级评论',
  `to_user_id` BIGINT NOT NULL DEFAULT 0 COMMENT '被回复用户ID',
  `like_count` BIGINT NOT NULL DEFAULT 0 COMMENT '评论点赞数',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `deleted_at` DATETIME DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (`id`),
  KEY `idx_video_root_created` (`video_id`, `root_id`, `created_at` DESC),
  KEY `idx_user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='评论表';
```

#### 表：`t_comment_like`

```sql
CREATE TABLE `t_comment_like` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `comment_id` BIGINT NOT NULL COMMENT '评论ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_comment` (`user_id`, `comment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='评论点赞表';
```

### 2.6 消息通知体系

#### 表：`t_message`

```sql
CREATE TABLE `t_message` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `receiver_id` BIGINT NOT NULL COMMENT '接收者ID',
  `sender_id` BIGINT NOT NULL DEFAULT 0 COMMENT '发送者ID,系统消息为0',
  `type` TINYINT NOT NULL COMMENT '消息类型:1点赞视频,2评论视频,3回复评论,4新增粉丝,5系统通知,6视频处理结果',
  `related_id` BIGINT NOT NULL DEFAULT 0 COMMENT '关联业务ID(视频ID/评论ID/用户ID)',
  `content` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '消息内容',
  `is_read` TINYINT NOT NULL DEFAULT 0 COMMENT '是否已读:0未读,1已读',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_receiver_read_created` (`receiver_id`, `is_read`, `created_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='消息通知表';
```

## 三、Redis 设计补充

|业务场景|Key|类型|说明|
|---|---|---|---|
|用户信息缓存|`user:info:{uid}`|Hash|昵称、头像、状态|
|用户统计缓存|`user:stats:{uid}`|Hash|关注数、粉丝数、作品数|
|关注集合|`following:{uid}`|Set|用于快速判断是否关注|
|粉丝集合|`followers:{uid}`|Set|用于关注流分发与粉丝列表|
|视频计数缓存|`video:stats:{vid}`|Hash|点赞数、评论数、收藏数、播放量|
|视频点赞集合|`video:liked:{vid}`|Set|点赞幂等控制|
|视频收藏集合|`video:favorited:{vid}`|Set|收藏幂等控制|
|评论点赞集合|`comment:liked:{cid}`|Set|评论点赞幂等控制|
|关注流收件箱|`feed:inbox:{uid}`|ZSet|当前用户关注流候选视频|
|作者发件箱|`feed:outbox:{uid}`|ZSet|大 V 发件箱|
|消息未读计数|`msg:unread:{uid}`|Hash|按消息类型计数|
|JWT 黑名单|`jwt:blacklist:{token}`|String|退出登录后失效 Token|

## 四、核心接口文档

全局响应格式：

```json
{
  "code": 0,
  "msg": "success",
  "data": {}
}
```

鉴权方式：

- Header: `Authorization: Bearer <JWT_TOKEN>`
- 除注册、登录外，其余接口均需鉴权
- 当前开发环境 JWT 使用 HS256，固定密钥为 `tiktide-system`

### 4.1 用户与鉴权

|接口名称|Method|Path|说明|
|---|---|---|---|
|注册|POST|`/api/v1/user/register`|参数：`username`、`password`|
|登录|POST|`/api/v1/user/login`|返回：`token`、用户基础信息|
|退出登录|POST|`/api/v1/user/logout`|将当前 Token 放入黑名单|
|获取个人资料|GET|`/api/v1/user/profile`|返回当前用户资料与统计数据|
|修改个人资料|PUT|`/api/v1/user/profile`|参数：`nickname`、`avatar_url`、`signature`、`gender`、`birthday`|
|修改密码|PUT|`/api/v1/user/password`|参数：`old_password`、`new_password`|
|获取他人主页|GET|`/api/v1/user/{uid}`|返回他人资料、统计信息、是否已关注、是否互关|

### 4.2 关注关系

|接口名称|Method|Path|说明|
|---|---|---|---|
|关注/取关用户|POST|`/api/v1/relation/action`|参数：`to_user_id`、`action_type(1关注,2取关)`|
|获取关注列表|GET|`/api/v1/relation/following/{uid}`|支持游标分页|
|获取粉丝列表|GET|`/api/v1/relation/followers/{uid}`|返回是否互关信息|

### 4.3 视频发布与草稿箱

|接口名称|Method|Path|说明|
|---|---|---|---|
|获取上传凭证|POST|`/api/v1/video/upload-credential`|返回：`upload_url`、`object_key`、`upload_method`、过期时间|
|发布视频|POST|`/api/v1/video/publish`|参数：`object_key`、`title`、`hashtag_ids`、`hashtag_names`、`allow_comment`、`visibility`|
|获取视频详情|GET|`/api/v1/video/{vid}`|返回视频信息、作者信息、互动状态、统计数据|
|获取视频多码率资源|GET|`/api/v1/video/{vid}/resources`|返回 480p/720p/1080p 资源列表|
|保存草稿|POST|`/api/v1/draft`|参数：`object_key`、`cover_url`、`title`、`tag_names`、`allow_comment`、`visibility`|
|草稿箱列表|GET|`/api/v1/draft/list`|返回当前用户草稿列表|
|删除草稿|DELETE|`/api/v1/draft/{id}`|删除指定草稿|

### 4.4 关注流与话题

|接口名称|Method|Path|说明|
|---|---|---|---|
|创建话题|POST|`/api/v1/hashtag`|参数：`name`。若话题已存在则直接返回已有话题|
|获取关注流|GET|`/api/v1/feed/following`|参数：`cursor`、`limit`，返回 `next_cursor`|
|获取话题详情|GET|`/api/v1/hashtag/{hid}`|返回话题信息|
|获取话题下视频|GET|`/api/v1/hashtag/{hid}/videos`|参数：`cursor(RFC3339 时间)`、`limit`|

### 4.5 互动功能

|接口名称|Method|Path|说明|
|---|---|---|---|
|视频点赞/取消|POST|`/api/v1/interact/like`|参数：`video_id`、`action_type(1点赞,2取消)`|
|视频收藏/取消|POST|`/api/v1/interact/favorite`|参数：`video_id`、`action_type(1收藏,2取消)`|
|我的收藏列表|GET|`/api/v1/interact/favorite/list`|支持游标分页|
|发表评论|POST|`/api/v1/interact/comment/publish`|参数：`video_id`、`content`、`parent_id`、`root_id`、`to_user_id`|
|获取评论列表|GET|`/api/v1/interact/comment/list`|参数：`video_id`、`root_id`、`cursor`、`limit`|
|评论点赞/取消|POST|`/api/v1/interact/comment/like`|参数：`comment_id`、`action_type(1点赞,2取消)`|

### 4.6 消息通知

|接口名称|Method|Path|说明|
|---|---|---|---|
|获取未读消息数|GET|`/api/v1/message/unread-count`|优先读 Redis|
|获取消息列表|GET|`/api/v1/message/list`|参数：`type`、`cursor`、`limit`|
|标记消息已读|POST|`/api/v1/message/read`|参数：`msg_id` 或 `type`|

## 五、接口与表结构的一致性说明

### 5.1 发布接口与视频表

- `POST /api/v1/video/publish` 对应写入 `t_video`
- 发布接口写入的是原始对象 `object_key`
- `source_url` 由当前 OSS `bucket + endpoint + object_key` 拼接得到
- 真正的播放资源写入 `t_video_resource`
- `cover_url` 由转码任务截帧后回写

### 5.2 关注流与视频可见性

Feed 服务只消费满足以下条件的视频：

- `visibility = 1`
- `audit_status = 1`
- `transcode_status = 2`

### 5.3 通知与互动解耦

- 点赞、评论、关注事件先完成主业务
- 通知通过 Kafka 异步生成，不阻塞主链路
- `t_message` 用于消息详情，`msg:unread:{uid}` 用于未读数快速读取

### 5.4 当前用户模块实现说明

- 当前用户与鉴权模块由 Gin HTTP 接口直接对外暴露
- 数据落库通过 GORM Repository 完成，不再使用 `database/sql` 直接操作
- 退出登录通过 Redis 黑名单失效 Token

### 5.5 当前关注关系模块实现说明

- 当前关注关系模块已独立拆分为 `relation` 模块，不与用户资料写逻辑耦合
- 关注与取关统一通过 `POST /api/v1/relation/action` 处理，便于后续接入通知与 Feed 分发
- 关注列表与粉丝列表均采用 `cursor + limit` 分页，当前 `cursor` 使用关系表自增 ID
- 用户主页接口 `GET /api/v1/user/{uid}` 已接入关注态查询，可返回 `is_followed` 与 `is_mutual`
- 关注关系写入时同步维护 `t_user_stats.follow_count` 与 `t_user_stats.follower_count`

### 5.6 当前话题模块实现说明

- 当前话题创建、话题详情与话题视频列表接口已经由 Video 模块统一提供
- 发布视频时支持继续传入 `hashtag_ids`
- 发布视频时也支持传入 `hashtag_names`，后端会自动创建不存在的话题并建立关联
- 当前话题视频列表只返回 `visibility=1` 且 `audit_status=1` 的视频

### 5.7 当前草稿箱模块实现说明

- 当前草稿箱已经支持保存草稿、查询当前用户草稿列表、删除草稿
- 草稿箱只保存视频发布前的元数据快照，不参与转码与 Feed 分发
- `tag_names` 采用逗号分隔字符串存储，便于后续与话题模块继续兼容

## 六、当前版本不纳入本表结构与接口文档的能力

为保证项目边界清晰，本版本不设计以下能力：

- 推荐流接口与推荐特征表
- 搜索、热搜、ES 索引结构
- 分享短链、二维码
- 直播推拉流
- 审核后台与运营管理后台

这类能力可作为后续二期扩展，不放在当前版本中承诺实现。
