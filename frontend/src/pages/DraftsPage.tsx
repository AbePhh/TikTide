import { useEffect, useMemo, useState } from "react";

import { deleteDraft, listDrafts, publishVideo, saveDraft } from "../api/client";
import { PageSectionHeader } from "../components/PageSectionHeader";
import { useAuth } from "../stores/auth";
import type { DraftData } from "../types/api";
import { formatRelativeTime } from "../utils/format";

type DraftEditorState = {
  id: string;
  objectKey: string;
  coverURL: string;
  title: string;
  topicInput: string;
  visibility: "public" | "private";
  allowComment: "allow" | "deny";
};

function visibilityText(value: number) {
  return value === 1 ? "公开" : "私密";
}

function buildEditorState(draft: DraftData): DraftEditorState {
  return {
    id: draft.id,
    objectKey: draft.object_key,
    coverURL: draft.cover_url,
    title: draft.title ?? "",
    topicInput: (draft.tag_names ?? "")
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean)
      .map((item) => `#${item}`)
      .join(" "),
    visibility: draft.visibility === 1 ? "public" : "private",
    allowComment: draft.allow_comment === 1 ? "allow" : "deny"
  };
}

function parseHashtagNames(raw: string) {
  return raw
    .split(/\s+/)
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => item.replace(/^#/, ""));
}

export function DraftsPage() {
  const { isAuthenticated } = useAuth();
  const [items, setItems] = useState<DraftData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [busyDraftID, setBusyDraftID] = useState<string | null>(null);
  const [editingDraft, setEditingDraft] = useState<DraftEditorState | null>(null);
  const [editorBusy, setEditorBusy] = useState(false);
  const [editorError, setEditorError] = useState<string | null>(null);
  const [editorSuccess, setEditorSuccess] = useState<string | null>(null);

  const parsedHashtags = useMemo(() => parseHashtagNames(editingDraft?.topicInput ?? ""), [editingDraft?.topicInput]);

  useEffect(() => {
    if (!isAuthenticated) {
      setItems([]);
      return;
    }

    let alive = true;
    setLoading(true);
    setError(null);

    listDrafts()
      .then((result) => {
        if (alive) {
          setItems(result.items);
        }
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

  async function handleDelete(id: string) {
    try {
      setBusyDraftID(id);
      await deleteDraft(id);
      setItems((current) => current.filter((item) => item.id !== id));
      if (editingDraft?.id === id) {
        setEditingDraft(null);
      }
    } catch (err) {
      if (err instanceof Error) {
        setError(err.message);
      }
    } finally {
      setBusyDraftID(null);
    }
  }

  function openEditor(draft: DraftData) {
    setEditingDraft(buildEditorState(draft));
    setEditorError(null);
    setEditorSuccess(null);
  }

  function closeEditor() {
    setEditingDraft(null);
    setEditorError(null);
    setEditorSuccess(null);
  }

  function updateEditor(patch: Partial<DraftEditorState>) {
    setEditingDraft((current) => (current ? { ...current, ...patch } : current));
  }

  async function handleUpdateDraft() {
    if (!editingDraft) {
      return;
    }

    setEditorBusy(true);
    setEditorError(null);
    setEditorSuccess(null);
    try {
      const updated = await saveDraft({
        draft_id: editingDraft.id,
        object_key: editingDraft.objectKey,
        cover_url: editingDraft.coverURL,
        title: editingDraft.title,
        tag_names: parsedHashtags.join(","),
        allow_comment: editingDraft.allowComment === "allow" ? 1 : 0,
        visibility: editingDraft.visibility === "public" ? 1 : 0
      });

      setItems((current) => current.map((item) => (item.id === updated.id ? updated : item)));
      setEditingDraft(buildEditorState(updated));
      setEditorSuccess("草稿已更新。");
    } catch (err) {
      if (err instanceof Error) {
        setEditorError(err.message);
      } else {
        setEditorError("草稿更新失败。");
      }
    } finally {
      setEditorBusy(false);
    }
  }

  async function handlePublishDraft() {
    if (!editingDraft) {
      return;
    }
    if (!editingDraft.title.trim()) {
      setEditorError("请先填写标题。");
      return;
    }

    setEditorBusy(true);
    setEditorError(null);
    setEditorSuccess(null);
    try {
      await publishVideo({
        object_key: editingDraft.objectKey,
        title: editingDraft.title.trim(),
        hashtag_ids: [],
        hashtag_names: parsedHashtags,
        allow_comment: editingDraft.allowComment === "allow" ? 1 : 0,
        visibility: editingDraft.visibility === "public" ? 1 : 0
      });
      await deleteDraft(editingDraft.id);
      setItems((current) => current.filter((item) => item.id !== editingDraft.id));
      closeEditor();
    } catch (err) {
      if (err instanceof Error) {
        setEditorError(err.message);
      } else {
        setEditorError("发布失败。");
      }
    } finally {
      setEditorBusy(false);
    }
  }

  return (
    <>
      <section className="page-block">
        <PageSectionHeader title="草稿箱" subtitle="草稿文件已上传，当前只是在未发布状态。这里支持直接预览、继续编辑和删除草稿。" />
        {!isAuthenticated ? <div className="panel panel-roomy">请先登录后查看草稿。</div> : null}
        {loading ? <div className="panel panel-roomy">正在加载草稿...</div> : null}
        {error ? <div className="panel panel-roomy">{error}</div> : null}

        <div className="draft-grid">
          {items.map((draft) => (
            <article key={draft.id} className="draft-card">
              <div className="draft-cover draft-cover-preview">
                {draft.cover_url ? (
                  <img className="draft-cover-image" src={draft.cover_url} alt={draft.title || "草稿封面"} />
                ) : draft.source_url ? (
                  <video className="draft-cover-video" src={draft.source_url} muted loop playsInline preload="metadata" autoPlay />
                ) : (
                  <div className="draft-cover-empty">暂无封面</div>
                )}
                <div className="draft-cover-mask" />
              </div>
              <div className="draft-body">
                <div className="draft-title">{draft.title || "未命名草稿"}</div>
                <div className="draft-meta">
                  <span>{formatRelativeTime(draft.updated_at)}</span>
                  <span>{visibilityText(draft.visibility)}</span>
                </div>
                <div className="draft-tags">{draft.tag_names || "无话题"}</div>
                <div className="draft-actions">
                  <button className="ghost-button draft-action-button" type="button" onClick={() => openEditor(draft)}>
                    继续编辑
                  </button>
                  <button className="ghost-button draft-action-button draft-delete-button" type="button" onClick={() => void handleDelete(draft.id)} disabled={busyDraftID === draft.id}>
                    删除草稿
                  </button>
                </div>
              </div>
            </article>
          ))}
          {!loading && isAuthenticated && items.length === 0 ? <div className="panel panel-roomy">当前草稿箱为空。</div> : null}
        </div>
      </section>

      {editingDraft ? (
        <div className="auth-modal-backdrop" onClick={closeEditor}>
          <div className="auth-modal panel panel-roomy draft-editor-modal" onClick={(event) => event.stopPropagation()}>
            <div className="relation-modal-header">
              <div>
                <h3>继续编辑草稿</h3>
                <p>这里直接编辑已上传草稿的标题、话题、权限与评论设置，无需重新上传文件。</p>
              </div>
              <button className="auth-close-button" type="button" onClick={closeEditor}>
                ×
              </button>
            </div>

            <div className="draft-editor-layout">
              <div className="draft-editor-preview">
                {editingDraft.coverURL ? (
                  <img className="draft-editor-cover-image" src={editingDraft.coverURL} alt={editingDraft.title || "草稿封面"} />
                ) : (
                  <video className="draft-editor-video" src={editingDraft.objectKey ? items.find((item) => item.id === editingDraft.id)?.source_url : ""} controls playsInline />
                )}
              </div>

              <div className="upload-form">
                <label className="form-field">
                  <span>标题</span>
                  <input value={editingDraft.title} onChange={(event) => updateEditor({ title: event.target.value })} placeholder="给作品写一个标题" />
                </label>
                <label className="form-field">
                  <span>话题</span>
                  <input value={editingDraft.topicInput} onChange={(event) => updateEditor({ topicInput: event.target.value })} placeholder="#城市记录 #生活方式" />
                </label>
                <div className="form-row">
                  <label className="form-field">
                    <span>可见范围</span>
                    <select value={editingDraft.visibility} onChange={(event) => updateEditor({ visibility: event.target.value as "public" | "private" })}>
                      <option value="public">公开</option>
                      <option value="private">私密</option>
                    </select>
                  </label>
                  <label className="form-field">
                    <span>允许评论</span>
                    <select value={editingDraft.allowComment} onChange={(event) => updateEditor({ allowComment: event.target.value as "allow" | "deny" })}>
                      <option value="allow">允许</option>
                      <option value="deny">禁止</option>
                    </select>
                  </label>
                </div>

                <div className="upload-meta-card">
                  <div className="upload-meta-item">
                    <span className="upload-meta-label">当前 object_key</span>
                    <span className="upload-meta-value upload-meta-value-break">{editingDraft.objectKey}</span>
                  </div>
                </div>

                {editorError ? <div className="form-error">{editorError}</div> : null}
                {editorSuccess ? <div className="form-success">{editorSuccess}</div> : null}

                <div className="form-actions">
                  <button className="ghost-button" type="button" onClick={() => void handleUpdateDraft()} disabled={editorBusy}>
                    更新草稿
                  </button>
                  <button className="primary-button" type="button" onClick={() => void handlePublishDraft()} disabled={editorBusy}>
                    发布作品
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}
