import { useEffect, useState } from "react";

import { getUnreadCount, listMessages, markMessageRead } from "../api/client";
import { PageSectionHeader } from "../components/PageSectionHeader";
import { useAuth } from "../stores/auth";
import type { MessageData } from "../types/api";
import { formatRelativeTime } from "../utils/format";

const messageTypeMap: Record<number, { key: "like" | "comment" | "reply" | "follow" | "system"; title: string }> = {
  1: { key: "like", title: "点赞通知" },
  2: { key: "comment", title: "评论通知" },
  3: { key: "reply", title: "回复提醒" },
  4: { key: "follow", title: "新粉丝通知" },
  5: { key: "system", title: "系统通知" },
  6: { key: "system", title: "视频处理通知" }
};

export function MessagesPage() {
  const { isAuthenticated } = useAuth();
  const [items, setItems] = useState<MessageData[]>([]);
  const [unreadMap, setUnreadMap] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isAuthenticated) {
      setItems([]);
      setUnreadMap({});
      return;
    }

    let alive = true;
    setLoading(true);
    setError(null);

    Promise.all([listMessages({ limit: 20 }), getUnreadCount()])
      .then(([messageData, unreadData]) => {
        if (!alive) {
          return;
        }
        setItems(messageData.items);
        setUnreadMap(unreadData.items);
      })
      .catch((err: Error) => {
        if (alive) {
          setError(err.message);
        }
      })
      .finally(() => {
        if (alive) {
          setLoading(false);
        }
      });

    return () => {
      alive = false;
    };
  }, [isAuthenticated]);

  async function handleRead(message: MessageData) {
    if (message.is_read === 1) {
      return;
    }
    try {
      await markMessageRead({ msg_id: message.id });
      setItems((current) =>
        current.map((item) => (item.id === message.id ? { ...item, is_read: 1 } : item))
      );
      setUnreadMap((current) => {
        const key = String(message.type);
        return {
          ...current,
          [key]: Math.max((current[key] ?? 1) - 1, 0)
        };
      });
    } catch {
      // no-op
    }
  }

  return (
    <section className="page-block">
      <PageSectionHeader
        title="消息中心"
        subtitle="已接入未读数、消息列表和单条已读。当前使用真实后端 message 接口。"
      />
      {!isAuthenticated ? <div className="panel panel-roomy">请先使用顶部 Demo 登录后查看消息。</div> : null}
      {loading ? <div className="panel panel-roomy">正在加载消息...</div> : null}
      {error ? <div className="panel panel-roomy">{error}</div> : null}
      <div className="message-stack">
        {items.map((item) => {
          const config = messageTypeMap[item.type] ?? messageTypeMap[5];
          const unreadCount = unreadMap[String(item.type)] ?? 0;
          return (
            <article
              key={item.id}
              className={`message-card ${item.is_read === 1 ? "message-card-read" : ""}`}
              onClick={() => void handleRead(item)}
            >
              <div className={`message-type message-type-${config.key}`}>{config.title.slice(0, 1)}</div>
              <div className="message-content">
                <div className="message-title-row">
                  <strong>{config.title}</strong>
                  <span>{formatRelativeTime(item.created_at)}</span>
                </div>
                <p>{item.content}</p>
                <div className="message-submeta">类型 {item.type} · 当前分类未读 {unreadCount}</div>
              </div>
              {item.is_read === 0 ? <span className="message-unread-dot" /> : null}
            </article>
          );
        })}
        {!loading && isAuthenticated && items.length === 0 ? (
          <div className="panel panel-roomy">当前还没有消息。</div>
        ) : null}
      </div>
    </section>
  );
}
