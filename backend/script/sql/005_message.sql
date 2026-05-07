CREATE TABLE IF NOT EXISTS `t_message` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `receiver_id` BIGINT NOT NULL COMMENT '接收者用户ID',
  `sender_id` BIGINT NOT NULL DEFAULT 0 COMMENT '发送者用户ID，系统消息为0',
  `type` TINYINT NOT NULL COMMENT '消息类型，6=视频处理结果',
  `related_id` BIGINT NOT NULL DEFAULT 0 COMMENT '关联业务ID，例如视频ID',
  `content` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '消息内容',
  `is_read` TINYINT NOT NULL DEFAULT 0 COMMENT '是否已读:0未读,1已读',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_receiver_type_id` (`receiver_id`, `type`, `id` DESC),
  KEY `idx_receiver_id` (`receiver_id`, `id` DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='站内消息表';
