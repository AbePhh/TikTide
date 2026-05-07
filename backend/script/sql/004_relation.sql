CREATE TABLE IF NOT EXISTS `t_relation` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `user_id` BIGINT NOT NULL COMMENT '关注者ID',
  `follow_id` BIGINT NOT NULL COMMENT '被关注者ID',
  `is_mutual` TINYINT NOT NULL DEFAULT 0 COMMENT '是否互关:0否,1是',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_follow` (`user_id`, `follow_id`),
  KEY `idx_follow_id` (`follow_id`),
  KEY `idx_user_id_id` (`user_id`, `id` DESC),
  KEY `idx_follow_id_id` (`follow_id`, `id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='关注关系表';
