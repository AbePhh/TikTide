# TikTide Frontend API Gaps

本文档只记录当前 `frontend` 与 `backend` 的真实对接情况，不写“理想中应该有”的假接口，也不把前端未接和后端缺失混在一起。
更新时间：2026-05-07

## 已完成对接

### 认证与个人信息

- 已接 `POST /api/v1/user/register`
- 已接 `POST /api/v1/user/login`
- 已接 `POST /api/v1/user/logout`
- 已接 `GET /api/v1/user/profile`
- 已接 `PUT /api/v1/user/profile`
- 已接 `PUT /api/v1/user/password`
- 已接 `PUT /api/v1/user/username`

### 关注关系

- 已接 `POST /api/v1/relation/action`
- 已接 `GET /api/v1/relation/following/:uid`
- 已接 `GET /api/v1/relation/followers/:uid`
- 个人主页中“关注数 / 粉丝数”已支持弹窗查看用户列表

### Feed 与视频播放

- 已接 `GET /api/v1/feed/following`
- 已接 `GET /api/v1/feed/recommend`
- 已接 `GET /api/v1/video/:vid/resources`
- 已接 `POST /api/v1/video/play/report`
- 推荐流已实现：
  - 首屏先拉少量数据
  - 滑到缓冲阈值自动预取下一批
  - 前端本地保留小缓冲区
  - 实际播放超过 2 秒才上报播放
  - 同一用户同一视频 30 分钟内去重

### 互动

- 已接 `POST /api/v1/interact/like`
- 已接 `POST /api/v1/interact/favorite`
- 已接 `GET /api/v1/interact/favorite/list`
- 已接 `POST /api/v1/interact/comment/publish`
- 已接 `GET /api/v1/interact/comment/list`
- 已接 `POST /api/v1/interact/comment/like`
- 评论区已改为右侧抽屉式交互

### 上传与草稿

- 已接 `POST /api/v1/video/upload-credential`
- 已接 `POST /api/v1/video/publish`
- 已接 `POST /api/v1/draft`
- 已接 `GET /api/v1/draft/:id`
- 已接 `GET /api/v1/draft/list`
- 已接 `DELETE /api/v1/draft/:id`
- 草稿支持继续编辑
- 草稿发布后自动删除对应草稿记录

### 用户主页与作品

- 已接 `GET /api/v1/user/:uid`
- 已接 `GET /api/v1/user/:uid/videos`
- 当前已支持：
  - 我的主页展示本人作品
  - 发现页跳转到创作者主页
  - 创作者主页展示资料与作品列表

### 发现页

- 已接 `GET /api/v1/feed/recommend` 作为“热门视频”数据源
- 已接 `GET /api/v1/hashtag/:hid`
- 已接 `GET /api/v1/hashtag/:hid/videos`
- 已接 `GET /api/v1/hashtag/hot`
- 当前发现页真实接入部分：
  - 热门话题
  - 热门视频
  - 热门创作者
  - 右侧热度排行

### 搜索

- 已接 `GET /api/v1/search/all`
- 已接 `GET /api/v1/search/users`
- 已接 `GET /api/v1/search/hashtags`
- 已接 `GET /api/v1/search/videos`
- 当前已完成：
  - 顶部搜索框真实联想
  - 回车跳转独立搜索结果页
  - 搜索结果页按“视频 / 作者 / 话题”分栏展示
  - 作者结果可跳创作者主页
  - 话题结果可跳话题页

## 本次新增或调整

### 后端已存在并已被前端接入

- `GET /api/v1/search/all`
  - 用途：顶部搜索框联想
  - 当前前端使用位置：`Topbar`

- `GET /api/v1/search/users`
  - 用途：搜索结果页作者 tab
  - 当前前端使用位置：`SearchPage`

- `GET /api/v1/search/hashtags`
  - 用途：搜索结果页话题 tab
  - 当前前端使用位置：`SearchPage`

- `GET /api/v1/search/videos`
  - 用途：搜索结果页视频 tab
  - 当前前端使用位置：`SearchPage`

### 前端本次新增

- 新增独立搜索结果页：`/search?q=...`
- 顶部搜索框从静态 UI 改为真实联想搜索
- 搜索结果页不再占用右侧空栏，改为单栏宽布局

## 仍未完成但不能假接的能力

### 搜索增强能力

当前后端虽然已有基础搜索接口，但以下能力仍未提供，因此前端没有接：

- 热搜词接口
- 搜索高亮片段
- 搜索历史
- 自动补全专用接口
- 搜索结果综合排序解释信息

建议后续接口：

- `GET /api/v1/search/hot`
- `GET /api/v1/search/suggest?q=`
- `GET /api/v1/search/history`

### 分类体系

当前前端需要的能力：

- 分类标签列表
- 按分类筛视频
- 分类页或分类聚合流

当前后端现状：

- 没有分类模型
- 没有分类关联关系
- 没有分类查询接口

因此当前处理：

- 发现页已删除这部分假分类 UI，不再做伪功能占位

建议后端补充：

- 分类表
- 视频与分类关联
- `GET /api/v1/category/list`
- `GET /api/v1/category/:cid/videos`

### 完整发现聚合接口

当前发现页仍是前端自行拼装：

- 热门话题：`/api/v1/hashtag/hot`
- 热门视频：`/api/v1/feed/recommend`
- 热门创作者：由推荐流作者信息前端聚合

当前问题：

- 需要多次请求
- 排序口径分散
- 热门创作者不是后端统一榜单

后续如果需要统一，可以新增：

- `GET /api/v1/discover`

但当前并非必须，已经可以真实运行。

## 需要注意的真实边界

### 搜索联想当前不支持

- 热词榜
- 拼音联想
- 搜索历史
- 高亮命中文案

这不是前端没做，而是当前后端接口尚未提供对应颗粒度。

### 搜索视频结果当前行为

当前 `/api/v1/search/videos` 返回的是视频卡片信息，不是“独立视频播放页”能力。

因此当前前端处理：

- 先展示搜索结果卡片
- 点击作者跳创作者主页
- 点击话题跳话题页
- 暂未把搜索结果直接接到沉浸式视频播放定位

如果后续要做“点击搜索结果直接进入指定视频播放位”，建议后端和前端一起补独立视频详情路由能力。

## 当前建议的后续优先级

### P0

- 搜索结果点击后进入指定视频播放定位
- 搜索热词接口

### P1

- 搜索建议专用接口
- 搜索历史
- 发现页聚合接口

### P2

- 搜索高亮
- 分类体系
- 更细的搜索排序与运营配置
