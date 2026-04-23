CREATE TABLE IF NOT EXISTS `t_video` (
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

CREATE TABLE IF NOT EXISTS `t_video_resource` (
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

CREATE TABLE IF NOT EXISTS `t_hashtag` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(64) NOT NULL COMMENT '话题名称(不含#)',
  `use_count` BIGINT NOT NULL DEFAULT 0 COMMENT '被引用次数',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='话题表';

CREATE TABLE IF NOT EXISTS `t_video_hashtag` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `video_id` BIGINT NOT NULL COMMENT '视频ID',
  `hashtag_id` BIGINT NOT NULL COMMENT '话题ID',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_video_hashtag` (`video_id`, `hashtag_id`),
  KEY `idx_hashtag_id` (`hashtag_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='视频话题关联表';
