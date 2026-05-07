# Tiktide 推荐流（TF-IDF）接入改造说明

## 1. 目的

当前后端已经实现的是“关注流”闭环，核心接口是：

- `GET /api/v1/feed/following`

当前实现方式是：

- 小 V：发视频时直接推送到粉丝 inbox
- 大 V：写 author outbox，读关注流时再拉取 merge

这套机制适合“关注关系驱动”的时间线，不适合“推荐流”。

如果你准备新增“推荐流”，并且第一阶段采用 **TF-IDF 内容推荐模式**，那么整体上不建议直接复用当前 `feed:inbox/outbox` 机制，而应该新增一条独立的推荐链路。

原因很简单：

- 关注流的排序核心是“发布时间”
- TF-IDF 推荐流的排序核心是“用户兴趣向量”和“视频内容向量”的相似度
- 两者的召回来源、缓存结构、刷新策略、分页游标都不同

所以建议是：

- 保留现有 `following feed`
- 新增独立的 `recommend feed`

---

## 2. 当前后端现状

结合你现有代码，当前与推荐流相关的已有能力如下。

### 2.1 已有数据基础

你现在已经有这些表和字段可以作为推荐基础：

- `t_video`
  - `id`
  - `user_id`
  - `title`
  - `cover_url`
  - `source_url`
  - `allow_comment`
  - `visibility`
  - `transcode_status`
  - `audit_status`
  - `play_count`
  - `like_count`
  - `comment_count`
  - `favorite_count`
  - `created_at`
- `t_video_hashtag`
  - 视频和话题关联
- `t_hashtag`
  - `name`
  - `use_count`
- `t_like`
- `t_favorite`
- `t_comment`
- `t_relation`
  - 可用于“降低重复推荐同一作者”或“增强已关注作者内容”

### 2.2 已有服务能力

你当前已有这些可复用能力：

- `videoService.GetVideoDetail(...)`
- `userRepo.GetByID(...)`
- `relationService.GetRelationState(...)`
- `interactRepo.HasLikedVideo(...)`
- `interactRepo.HasFavoritedVideo(...)`

这些能力已经足够支持“推荐流返回展示层数据”。

### 2.3 当前缺失的核心能力

你目前缺的不是“视频展示接口”，而是推荐系统特有的这几层：

- 用户兴趣画像
- 视频内容向量/关键词权重
- 推荐候选集合
- 推荐打分
- 推荐结果缓存
- 推荐结果翻页游标
- 推荐曝光/点击/播放反馈闭环

---

## 3. 推荐流方案建议

如果你第一阶段用 TF-IDF，我建议采用：

### 3.1 总体架构

推荐流拆成四层：

1. 内容特征层  
为每条视频生成 TF-IDF 特征。

2. 用户兴趣层  
根据用户点赞、收藏、评论、播放等行为，累积用户兴趣关键词权重。

3. 候选召回层  
从最近公开视频中召回一批视频候选。

4. 排序层  
使用“用户兴趣向量 vs 视频内容向量”的相似度打分，再叠加热度、去重、新鲜度做最终排序。

---

## 4. 数据库需要修改的地方

这是最重要的部分。

如果你只靠现有 `t_video.title + hashtag`，也能做一个最小版推荐流，但效果会很弱，而且后续难扩展。

建议新增以下表。

### 4.1 视频内容特征表

建议新增：

- `t_video_profile`

建议字段：

- `video_id bigint primary key`
- `author_user_id bigint not null`
- `title_terms_json json not null`
- `hashtag_terms_json json not null`
- `content_terms_json json null`
- `tfidf_vector_json json not null`
- `feature_version int not null default 1`
- `created_at datetime not null`
- `updated_at datetime not null`

作用：

- 保存视频的内容特征
- 支持后续重算
- 不直接依赖实时现算

说明：

- 第一阶段没有字幕/OCR/ASR 时，可以只用 `title + hashtag`
- 后续如果接入字幕抽取、OCR、ASR，再把词扩展进去

### 4.2 用户兴趣画像表

建议新增：

- `t_user_interest_profile`

建议字段：

- `user_id bigint primary key`
- `interest_terms_json json not null`
- `positive_terms_json json null`
- `negative_terms_json json null`
- `profile_version int not null default 1`
- `updated_at datetime not null`

作用：

- 保存用户当前兴趣关键词权重
- 在线推荐时直接读取，不用每次扫交互表重算

### 4.3 推荐曝光记录表

建议新增：

- `t_recommend_exposure`

建议字段：

- `id bigint primary key`
- `user_id bigint not null`
- `video_id bigint not null`
- `request_id varchar(64) not null`
- `position int not null`
- `scene varchar(32) not null`
- `score decimal(12,6) not null`
- `exposed_at datetime not null`

建议索引：

- `(user_id, exposed_at desc)`
- `(request_id)`
- `(user_id, video_id, exposed_at desc)`

作用：

- 防止短时间重复推荐
- 支持后续 CTR / 完播率分析
- 支持“为什么推荐这条”的回溯

### 4.4 推荐反馈行为表

如果你想后续做推荐效果优化，建议新增：

- `t_recommend_feedback`

建议字段：

- `id bigint primary key`
- `user_id bigint not null`
- `video_id bigint not null`
- `request_id varchar(64) not null`
- `action_type tinyint not null`
- `action_value int not null default 1`
- `created_at datetime not null`

说明：

- `action_type` 可定义为：
  - `1` 曝光后点击播放
  - `2` 有效播放
  - `3` 完播
  - `4` 点赞
  - `5` 收藏
  - `6` 评论
  - `7` 不感兴趣
  - `8` 跳过

### 4.5 可选：词典与 IDF 表

如果你希望 TF-IDF 更规范，建议再加：

- `t_term_dict`
- `t_term_idf`

建议字段：

- `t_term_dict`
  - `term_id bigint`
  - `term varchar(128)`
- `t_term_idf`
  - `term_id bigint`
  - `idf_score decimal(12,6)`
  - `doc_count bigint`
  - `updated_at datetime`

但这个不是第一阶段必须。

第一阶段也可以直接把词和权重都存 JSON。

---

## 5. 现有表建议补充的字段

### 5.1 t_video

建议补：

- `recommendable tinyint not null default 1`

作用：

- 是否允许进入推荐池
- 后续风控、人工运营、内容降权都会用到

还建议考虑补：

- `language varchar(16) null`
- `category_id bigint null`

如果后续想做垂类推荐，这两个字段会很有用。

### 5.2 t_comment / t_like / t_favorite

这些表结构不一定必须改，但建议确保有足够索引。

至少要有：

- `t_like(user_id, created_at desc)`
- `t_favorite(user_id, created_at desc)`
- `t_comment(user_id, created_at desc)`

因为用户兴趣画像重建时要高频读这些行为。

---

## 6. Redis 需要新增的内容

当前你已有：

- `feed:inbox:{user_id}`
- `feed:outbox:{user_id}`

推荐流不要复用它们。

建议新增：

- `feed:recommend:{user_id}`
- `feed:recommend:seen:{user_id}`

### 6.1 推荐结果缓存

`feed:recommend:{user_id}`

建议结构：

- `ZSET`
- `member = video_id`
- `score = 推荐分`

作用：

- 预先缓存一批推荐结果
- 支持快速分页

### 6.2 已曝光去重集合

`feed:recommend:seen:{user_id}`

建议结构：

- `ZSET` 或 `SET`

作用：

- 最近一段时间内不重复推同一视频
- 避免用户连续刷到重复内容

如果用 `ZSET`：

- `score = exposed_at`
- 可按时间裁剪

---

## 7. 后端服务层需要改的地方

### 7.1 新增独立推荐服务

当前 `internal/feed/service/service.go` 更偏向“关注流服务”。

不建议直接在里面硬塞推荐逻辑。

建议新增：

- `internal/recommend/service`

理由：

- 关注流和推荐流责任不同
- 推荐流未来会越来越复杂
- 独立服务便于测试、回放、灰度

建议定义：

- `RecommendService`

核心方法：

- `ListRecommend(ctx, userID, req)`
- `BuildUserProfile(ctx, userID)`
- `BuildVideoProfile(ctx, videoID)`
- `RefreshRecommendCache(ctx, userID)`
- `RecordExposure(ctx, userID, items, requestID)`

### 7.2 现有 feed handler 需要新增接口

建议新增接口：

- `GET /api/v1/feed/recommend`

不要替换现有：

- `GET /api/v1/feed/following`

推荐流和关注流应该并存。

### 7.3 app context 需要挂载 RecommendService

当前在：

- `internal/app/context.go`

里挂了：

- `VideoService`
- `FeedService`
- `InteractService`
- `MessageService`

这里要新增：

- `RecommendService`

### 7.4 feed handler 只读推荐结果，不做重计算

接口层不要现场全量算 TF-IDF。

否则：

- 请求慢
- 不稳定
- 不利于扩展

建议：

- handler 只调用 `RecommendService.ListRecommend(...)`
- 真正特征构建和缓存刷新放后台任务或懒更新逻辑里

---

## 8. 视频发布与转码流程需要改的地方

你现在视频发布流程大致是：

1. 上传成功
2. `video/publish`
3. 创建 `t_video`
4. 触发转码
5. 转码成功后进入 feed distribute

推荐流接入后，建议增加两步。

### 8.1 视频发布后创建初始内容特征

在 `video publish` 完成后，至少先根据：

- `title`
- `hashtag_names`

生成一份初始内容特征，写入 `t_video_profile`

### 8.2 转码成功后补充可推荐状态

当前公开视频只有转码成功、审核通过后才真正可见。

推荐流也要遵循同样约束。

建议转码成功后：

- 标记该视频可以进入推荐候选池
- 刷新相关召回缓存

但不要像关注流那样“硬推到所有人”。

---

## 9. 用户行为链路需要改的地方

TF-IDF 推荐的核心不只是视频内容，还要有用户兴趣。

所以这些行为发生时，推荐系统需要感知：

- 点赞视频
- 取消点赞
- 收藏视频
- 取消收藏
- 评论视频
- 播放视频
- 完播视频
- 跳过视频

你当前已有：

- 点赞/收藏/评论业务闭环

但当前更多是“业务统计”和“通知”，还没有“推荐反馈”。

建议：

### 9.1 interact service 增加画像更新钩子

位置：

- `internal/interact/service/service.go`

例如：

- 点赞成功后：增强该视频关键词权重到用户画像
- 收藏成功后：比点赞更高权重增强
- 评论成功后：中高权重增强

### 9.2 新增播放行为上报接口

你现在如果没有播放行为接口，推荐效果会很一般。

建议未来新增：

- `POST /api/v1/feed/recommend/exposure`
- `POST /api/v1/feed/recommend/feedback`

或者最简先做：

- `POST /api/v1/video/play/report`

否则系统只知道“点赞/收藏/评论”，不知道“看了但没互动”和“秒划走”。

这会严重影响用户兴趣建模。

---

## 10. TF-IDF 推荐第一阶段的最小实现建议

如果你想尽快上线，不要一上来做太复杂。

建议第一阶段只做下面这些。

### 10.1 视频向量来源

仅使用：

- 视频标题分词
- hashtag 名称分词

不做：

- 语音转文字
- OCR
- 封面图像标签

### 10.2 用户兴趣来源

仅使用近 30 天：

- 点赞视频
- 收藏视频
- 评论视频

其中权重建议：

- 收藏 = 3
- 评论 = 2
- 点赞 = 1

### 10.3 候选集来源

从最近 N 天的公开视频中选候选：

- `visibility = public`
- `audit_status = passed`
- `transcode_status = success`
- `recommendable = 1`

例如先取最近 7 天的 3000 条。

### 10.4 排序分数

建议：

`final_score = 内容相似度 * 0.7 + 热度分 * 0.2 + 新鲜度分 * 0.1`

热度分可由这些字段构成：

- `like_count`
- `favorite_count`
- `comment_count`
- `play_count`

### 10.5 去重规则

至少做：

- 已点赞的视频不再推荐
- 已收藏的视频不再推荐
- 自己发布的视频不推荐给自己
- 最近已曝光视频短时间内不重复推

---

## 11. SQL / 索引层面的建议

### 11.1 t_video

建议至少补这些索引：

- `(visibility, audit_status, transcode_status, created_at desc)`
- `(user_id, created_at desc)`
- `(recommendable, created_at desc)`

### 11.2 t_like

- `(user_id, created_at desc)`
- `(video_id, user_id)` 唯一索引

### 11.3 t_favorite

- `(user_id, created_at desc)`
- `(video_id, user_id)` 唯一索引

### 11.4 t_comment

- `(video_id, root_id, created_at desc)`
- `(user_id, created_at desc)`

### 11.5 新增特征表

`t_video_profile`

- `primary key(video_id)`
- `index(author_user_id)`
- `index(updated_at)`

`t_user_interest_profile`

- `primary key(user_id)`
- `index(updated_at)`

`t_recommend_exposure`

- `index(user_id, exposed_at desc)`
- `index(user_id, video_id, exposed_at desc)`
- `index(request_id)`

---

## 12. 接口层需要新增什么

### 12.1 必须新增

- `GET /api/v1/feed/recommend`

参数建议：

- `cursor`
- `limit`
- `scene` 可选，如 `default`

返回结构尽量复用你当前：

- `types.FeedVideoListData`

这样前端最省事。

### 12.2 建议新增

- `POST /api/v1/feed/recommend/feedback`

请求体建议：

- `video_id`
- `request_id`
- `action_type`
- `watch_duration_ms`

### 12.3 可后置

- `POST /api/v1/feed/recommend/refresh`

只给内部调试或管理端使用，不对普通用户开放。

---

## 13. 现有代码模块建议怎么改

### 13.1 internal/feed/service

当前保留为“关注流服务”即可。

建议：

- 不要把推荐逻辑揉进去
- 最多抽公共的 `buildFeedItem` 能力复用

### 13.2 internal/http/handler/feed_handler.go

新增：

- `ListRecommend`

保留：

- `ListFollowing`

### 13.3 internal/http/router/router.go

新增路由：

- `authenticated.GET("/feed/recommend", feedHandler.ListRecommend)`

### 13.4 internal/app/context.go

新增：

- `RecommendService`

并在 `New(...)` 里完成注入。

### 13.5 internal/video/service

发布视频成功后：

- 触发视频内容画像构建

### 13.6 internal/interact/service

点赞/收藏/评论成功后：

- 触发用户兴趣画像增量更新

---

## 14. 文档层面需要同步修改的地方

你现在的项目文档里如果要接入推荐流，建议同步补以下章节：

- Feed 流拆分为：
  - 关注流
  - 推荐流
- 推荐流排序策略说明
- 推荐画像与特征表设计
- 推荐反馈采集说明
- 推荐流缓存与降级方案

---

## 15. 我对你这个方案的建议

### 15.1 可以做，但不要把 TF-IDF 当终局

TF-IDF 非常适合你现在这个阶段：

- 实现成本低
- 解释性强
- 容易调试
- 不依赖复杂模型

但它的问题也很明显：

- 强依赖文本质量
- 对短标题视频不友好
- 无法很好理解隐式兴趣
- 容易推荐同质化标签内容

所以建议把 TF-IDF 定位成：

- 第一阶段可运行推荐流

而不是最终架构。

### 15.2 第一阶段不要改太多线上主链路

建议先新增，不替换：

- 不替换 `feed/following`
- 新增 `feed/recommend`

这样风险最低。

### 15.3 第一阶段不要急着做全量实时更新

最小可行建议：

- 发布视频时生成视频画像
- 用户点赞/收藏/评论时懒更新用户画像
- 用户拉推荐流时，如果缓存为空再重建一批

这样先把闭环跑起来。

---

## 16. 建议的实施优先级

### P0 必做

- 新增 `GET /api/v1/feed/recommend`
- 新增 `RecommendService`
- 新增 `t_video_profile`
- 新增 `t_user_interest_profile`
- 新增推荐结果 Redis key
- 基于 `title + hashtag` 生成视频向量
- 基于 `like/favorite/comment` 生成用户兴趣向量
- 推荐结果返回复用当前 `FeedVideoListData`

### P1 很建议做

- 新增 `t_recommend_exposure`
- 推荐去重缓存
- 最近曝光去重
- 热度分与新鲜度分融合
- 已点赞/已收藏过滤

### P2 后续做

- 播放时长反馈
- 跳过/不感兴趣反馈
- OCR/ASR/字幕特征
- 多路召回混排
- 运营干预

---

## 17. 结论

如果你要在当前 TikTide 后端上接入 TF-IDF 推荐流，**需要修改，而且修改点不只在接口层，也包括数据库、Redis、服务分层和行为反馈链路**。

最关键的结论是：

1. 不要复用当前关注流 `feed:inbox/outbox` 作为推荐流主体
2. 推荐流应新增独立服务和独立缓存
3. 数据库至少要新增：
   - `t_video_profile`
   - `t_user_interest_profile`
   - 建议再加 `t_recommend_exposure`
4. 第一阶段用 `title + hashtag` 做 TF-IDF 是合理的
5. 前期推荐接口建议独立为：
   - `GET /api/v1/feed/recommend`

如果你愿意，下一步我可以继续直接帮你输出第二份文档：

- 《Tiktide 推荐流（TF-IDF）数据库 DDL 与接口设计稿》

我可以把表结构、索引、接口请求响应、Redis key 命名、服务接口定义，直接按你当前项目代码风格写出来。  
