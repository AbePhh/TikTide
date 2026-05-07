import imageCompression from "browser-image-compression";
import * as qiniu from "qiniu-js";

export interface UploadProgress {
  total: number;
  loaded: number;
  percent: number;
}

export interface UploadTaskOptions {
  file: File;
  uploadToken: string;
  objectKey: string;
  onProgress?: (progress: UploadProgress) => void;
}

interface QiniuUploadError {
  code?: number | string;
  error?: string;
  message?: string;
  reqId?: string;
  key?: string;
  status?: number;
  data?: unknown;
}

const RESUME_KEY_PREFIX = "tiktide_upload_resume:";

function formatUploadError(error: unknown, objectKey: string) {
  const detail = (error ?? {}) as QiniuUploadError;
  const messageParts = [
    detail.message,
    detail.error,
    detail.code ? `code=${detail.code}` : "",
    detail.status ? `status=${detail.status}` : "",
    detail.reqId ? `reqId=${detail.reqId}` : "",
    detail.key ? `key=${detail.key}` : "",
    `objectKey=${objectKey}`
  ].filter(Boolean);

  const finalMessage = messageParts.length > 0 ? messageParts.join(" | ") : `upload failed | objectKey=${objectKey}`;
  return new Error(finalMessage);
}

export async function compressFileIfNeeded(file: File) {
  if (!file.type.startsWith("image/")) {
    return file;
  }

  return imageCompression(file, {
    maxSizeMB: 3,
    maxWidthOrHeight: 1920,
    useWebWorker: true
  });
}

export async function uploadFileWithQiniu(options: UploadTaskOptions) {
  const compressedFile = await compressFileIfNeeded(options.file);
  const resumeKey = RESUME_KEY_PREFIX + options.objectKey;

  console.info("[upload] start", {
    objectKey: options.objectKey,
    fileName: compressedFile.name,
    fileType: compressedFile.type,
    fileSize: compressedFile.size,
    tokenPreview: options.uploadToken ? `${options.uploadToken.slice(0, 24)}...` : null
  });

  const observable = qiniu.upload(compressedFile, options.objectKey, options.uploadToken, undefined, {
    uphost: ["up.qiniup.com"],
    useCdnDomain: false,
    forceDirect: true,
    concurrentRequestLimit: 4,
    chunkSize: 4
  });

  await new Promise<void>((resolve, reject) => {
    observable.subscribe({
      next(progressEvent) {
        const percent = progressEvent.total.percent ?? 0;
        const loaded = Math.round((progressEvent.total.loaded ?? 0) as number);
        const total = Math.round((progressEvent.total.size ?? compressedFile.size) as number);

        window.localStorage.setItem(
          resumeKey,
          JSON.stringify({
            objectKey: options.objectKey,
            fileName: compressedFile.name,
            size: compressedFile.size,
            lastPercent: percent
          })
        );

        options.onProgress?.({
          total,
          loaded,
          percent
        });
      },
      error(error) {
        console.error("[upload] qiniu error", {
          objectKey: options.objectKey,
          tokenPreview: options.uploadToken ? `${options.uploadToken.slice(0, 24)}...` : null,
          error
        });
        reject(formatUploadError(error, options.objectKey));
      },
      complete(result) {
        console.info("[upload] complete", {
          objectKey: options.objectKey,
          result
        });
        window.localStorage.removeItem(resumeKey);
        resolve();
      }
    });
  });
}
