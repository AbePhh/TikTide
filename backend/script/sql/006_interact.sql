CREATE TABLE IF NOT EXISTS `t_like` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `video_id` BIGINT NOT NULL COMMENT '视频ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_video` (`user_id`, `video_id`),
  KEY `idx_video_id` (`video_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频点赞表';

CREATE TABLE IF NOT EXISTS `t_favorite` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `video_id` BIGINT NOT NULL COMMENT '视频ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_video` (`user_id`, `video_id`),
  KEY `idx_user_created` (`user_id`, `created_at` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频收藏表';

CREATE TABLE IF NOT EXISTS `t_comment` (
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

CREATE TABLE IF NOT EXISTS `t_comment_like` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '用户ID',
  `comment_id` BIGINT NOT NULL COMMENT '评论ID',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_comment` (`user_id`, `comment_id`),
  KEY `idx_comment_id` (`comment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='评论点赞表';
