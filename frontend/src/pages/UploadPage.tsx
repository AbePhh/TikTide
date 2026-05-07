import { useMemo, useState } from "react";

import { createUploadCredential, publishVideo, saveDraft } from "../api/client";
import { PageSectionHeader } from "../components/PageSectionHeader";
import { useAuth } from "../stores/auth";
import { uploadFileWithQiniu } from "../utils/upload";
import { extractVideoCover } from "../utils/video";

function parseHashtagNames(raw: string) {
  return raw
    .split(/\s+/)
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => item.replace(/^#/, ""));
}

function buildDerivedObjectKey(sourceObjectKey: string, fileName: string) {
  const trimmed = sourceObjectKey.replace(/^\/+|\/+$/g, "");
  const dotIndex = trimmed.lastIndexOf(".");
  const base = dotIndex >= 0 ? trimmed.slice(0, dotIndex) : trimmed;
  return `${base}/${fileName}`;
}

const TOKEN_KEY = "tiktide_token";

export function UploadPage() {
  const { isAuthenticated } = useAuth();
  const [file, setFile] = useState<File | null>(null);
  const [title, setTitle] = useState("");
  const [topicInput, setTopicInput] = useState("");
  const [visibility, setVisibility] = useState<"public" | "private">("public");
  const [allowComment, setAllowComment] = useState<"allow" | "deny">("allow");
  const [objectKey, setObjectKey] = useState("");
  const [coverURL, setCoverURL] = useState("");
  const [coverObjectKey, setCoverObjectKey] = useState("");
  const [uploadCompleted, setUploadCompleted] = useState(false);
  const [status, setStatus] = useState("尚未开始上传");
  const [progressText, setProgressText] = useState("0%");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const parsedHashtags = useMemo(() => parseHashtagNames(topicInput), [topicInput]);
  const hasStoredToken = Boolean(window.localStorage.getItem(TOKEN_KEY));

  function getStoredToken() {
    return window.localStorage.getItem(TOKEN_KEY);
  }

  async function handlePrepareUpload() {
    if (!file) {
      setError("请先选择一个视频文件");
      setSuccessMessage(null);
      return;
    }

    const storedToken = getStoredToken();
    if (!storedToken) {
      setError("当前没有检测到登录凭证，请先重新登录后再上传");
      setSuccessMessage(null);
      setStatus("上传未开始");
      return;
    }

    setBusy(true);
    setError(null);
    setSuccessMessage(null);
    setProgressText("0%");
    setUploadCompleted(false);

    try {
      const credential = await createUploadCredential(file.name, file.type || "application/octet-stream");
      setObjectKey(credential.object_key);
      setCoverURL("");
      setCoverObjectKey("");
      setStatus(`已生成上传凭证，开始上传。object_key: ${credential.object_key}`);

      await uploadFileWithQiniu({
        file,
        uploadToken: credential.upload_token,
        objectKey: credential.object_key,
        onProgress(progress) {
          setProgressText(`${progress.percent.toFixed(1)}%`);
          setStatus(
            `上传中：${progress.percent.toFixed(1)}%，已完成 ${Math.round(progress.loaded / 1024 / 1024)}MB / ${Math.round(progress.total / 1024 / 1024)}MB`
          );
        }
      });

      const derivedCoverObjectKey = buildDerivedObjectKey(credential.object_key, "cover.jpg");
      const coverFile = await extractVideoCover(file);
      const coverCredential = await createUploadCredential("cover.jpg", coverFile.type || "image/jpeg", derivedCoverObjectKey);
      await uploadFileWithQiniu({
        file: coverFile,
        uploadToken: coverCredential.upload_token,
        objectKey: derivedCoverObjectKey
      });

      setCoverObjectKey(derivedCoverObjectKey);
      setCoverURL(URL.createObjectURL(coverFile));

      setUploadCompleted(true);
      setStatus(`文件上传成功，object_key: ${credential.object_key}`);
      setProgressText("100%");
      setSuccessMessage("上传成功，视频源文件和本地提取的封面都已经进入对象存储。现在可以继续保存草稿或直接发布。");
    } catch (err) {
      console.error("[upload-page] upload failed", err);
      setUploadCompleted(false);
      setSuccessMessage(null);
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("上传失败，请查看控制台日志");
      }
      setStatus("上传未完成");
    } finally {
      setBusy(false);
    }
  }

  async function handleSaveDraft() {
    if (!objectKey) {
      setError("请先完成文件上传，草稿保存依赖已生成的 object_key");
      setSuccessMessage(null);
      return;
    }
    if (!uploadCompleted) {
      setError("文件还没有成功上传完成，暂时不能保存草稿");
      setSuccessMessage(null);
      return;
    }

    const storedToken = getStoredToken();
    if (!storedToken) {
      setError("当前没有检测到登录凭证，请先重新登录后再保存草稿");
      setSuccessMessage(null);
      return;
    }

    setBusy(true);
    setError(null);
    setSuccessMessage(null);
    try {
      const result = await saveDraft({
        object_key: objectKey,
        cover_url: coverObjectKey,
        title,
        tag_names: parsedHashtags.join(","),
        allow_comment: allowComment === "allow" ? 1 : 0,
        visibility: visibility === "public" ? 1 : 0
      });
      setCoverURL(result.cover_url ?? "");
      setStatus("草稿保存成功");
      setSuccessMessage("草稿已经保存成功，你可以稍后继续编辑或直接发布。");
    } catch (err) {
      console.error("[upload-page] save draft failed", err);
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("草稿保存失败，请查看控制台日志");
      }
    } finally {
      setBusy(false);
    }
  }

  async function handlePublish() {
    if (!objectKey) {
      setError("请先完成文件上传");
      setSuccessMessage(null);
      return;
    }
    if (!uploadCompleted) {
      setError("文件还没有成功上传完成，暂时不能发布");
      setSuccessMessage(null);
      return;
    }
    if (!title.trim()) {
      setError("请先填写标题");
      setSuccessMessage(null);
      return;
    }

    const storedToken = getStoredToken();
    if (!storedToken) {
      setError("当前没有检测到登录凭证，请先重新登录后再发布");
      setSuccessMessage(null);
      return;
    }

    setBusy(true);
    setError(null);
    setSuccessMessage(null);
    try {
      const result = await publishVideo({
        object_key: objectKey,
        title: title.trim(),
        hashtag_ids: [],
        hashtag_names: parsedHashtags,
        allow_comment: allowComment === "allow" ? 1 : 0,
        visibility: visibility === "public" ? 1 : 0
      });
      setStatus(`发布成功，video_id=${result.video_id}，transcode_status=${result.transcode_status}`);
      setSuccessMessage("作品发布成功，系统已经进入后续转码处理流程。");
    } catch (err) {
      console.error("[upload-page] publish failed", err);
      if (err instanceof Error) {
        setError(err.message);
      } else {
        setError("发布失败，请查看控制台日志");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="page-block">
      <PageSectionHeader title="上传视频" subtitle="这里只处理新视频上传。草稿继续编辑请在草稿箱中直接完成，无需重新上传原文件。" />
      {!isAuthenticated ? <div className="panel panel-roomy">请先登录后再进行上传、保存草稿和发布操作。</div> : null}

      <section className="upload-shell panel panel-roomy">
        <div className="upload-dropzone">
          <div className="upload-drop-icon">↑</div>
          <div className="upload-drop-title">选择视频文件后，执行七牛分片上传</div>
          <div className="upload-drop-subtitle">服务端负责生成 object_key 与上传凭证，前端负责上传过程控制、进度反馈与断点续传。</div>

          <label className="form-field upload-filename-field">
            <span>视频文件</span>
            <div className="file-input-shell">
              <input
                className="file-input-native"
                type="file"
                accept="video/mp4,video/quicktime,video/*"
                onChange={(event) => {
                  setFile(event.target.files?.[0] ?? null);
                  setObjectKey("");
                  setCoverURL("");
                  setCoverObjectKey("");
                  setUploadCompleted(false);
                  setStatus("尚未开始上传");
                  setProgressText("0%");
                  setError(null);
                  setSuccessMessage(null);
                }}
              />
              <span className="file-input-button">选择文件</span>
              <span className={`file-input-name ${file ? "file-input-name-selected" : ""}`}>{file?.name || "未选择任何文件"}</span>
            </div>
          </label>

          <button className="ghost-button" type="button" onClick={() => void handlePrepareUpload()} disabled={busy || !isAuthenticated}>
            {busy ? "上传处理中..." : "开始上传"}
          </button>

          <div className="upload-status-card">
            <div className="upload-status">{status}</div>
            <div className="upload-progress-row">
              <span>上传进度</span>
              <strong>{progressText}</strong>
            </div>
          </div>

          {coverURL ? (
            <div className="upload-draft-cover-preview">
              <img src={coverURL} alt="草稿封面预览" />
            </div>
          ) : null}

          {successMessage ? <div className="form-success upload-feedback">{successMessage}</div> : null}
        </div>

        <form className="upload-form">
          <label className="form-field">
            <span>标题</span>
            <input value={title} onChange={(event) => setTitle(event.target.value)} placeholder="给作品写一个标题" />
          </label>
          <label className="form-field">
            <span>话题</span>
            <input value={topicInput} onChange={(event) => setTopicInput(event.target.value)} placeholder="#城市记录 #生活方式" />
          </label>
          <div className="form-row">
            <label className="form-field">
              <span>可见范围</span>
              <select value={visibility} onChange={(event) => setVisibility(event.target.value as "public" | "private")}>
                <option value="public">公开</option>
                <option value="private">私密</option>
              </select>
            </label>
            <label className="form-field">
              <span>允许评论</span>
              <select value={allowComment} onChange={(event) => setAllowComment(event.target.value as "allow" | "deny")}>
                <option value="allow">允许</option>
                <option value="deny">禁止</option>
              </select>
            </label>
          </div>

          {error ? <div className="form-error">{error}</div> : null}

          <div className="upload-meta-card">
            <div className="upload-meta-item">
              <span className="upload-meta-label">当前文件</span>
              <span className="upload-meta-value upload-meta-value-break">{file?.name || "未选择"}</span>
            </div>
            <div className="upload-meta-item">
              <span className="upload-meta-label">当前 object_key</span>
              <span className="upload-meta-value upload-meta-value-break">{objectKey || "未生成"}</span>
            </div>
            <div className="upload-meta-item">
              <span className="upload-meta-label">上传完成状态</span>
              <span className="upload-meta-value">{uploadCompleted ? "成功" : "未完成"}</span>
            </div>
            <div className="upload-meta-item">
              <span className="upload-meta-label">当前登录 token</span>
              <span className="upload-meta-value">{hasStoredToken ? "已存在" : "不存在"}</span>
            </div>
          </div>

          <div className="form-actions">
            <button className="ghost-button" type="button" onClick={() => void handleSaveDraft()} disabled={busy || !isAuthenticated}>
              保存草稿
            </button>
            <button className="primary-button" type="button" onClick={() => void handlePublish()} disabled={busy || !isAuthenticated}>
              发布作品
            </button>
          </div>
        </form>
      </section>
    </section>
  );
}
