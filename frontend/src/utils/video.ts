export async function extractVideoCover(file: File, seekToSeconds = 1): Promise<File> {
  const video = document.createElement("video");
  video.preload = "metadata";
  video.muted = true;
  video.playsInline = true;

  const objectURL = URL.createObjectURL(file);
  video.src = objectURL;

  await new Promise<void>((resolve, reject) => {
    video.onloadedmetadata = () => resolve();
    video.onerror = () => reject(new Error("视频元数据加载失败"));
  });

  const duration = Number.isFinite(video.duration) ? video.duration : 0;
  const targetTime = duration > 0 ? Math.min(seekToSeconds, Math.max(duration * 0.2, 0.1), Math.max(duration - 0.1, 0)) : 0;

  await new Promise<void>((resolve, reject) => {
    video.onseeked = () => resolve();
    video.onerror = () => reject(new Error("视频截帧失败"));
    video.currentTime = targetTime;
  });

  const width = video.videoWidth || 720;
  const height = video.videoHeight || 1280;
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;

  const context = canvas.getContext("2d");
  if (!context) {
    URL.revokeObjectURL(objectURL);
    throw new Error("浏览器不支持视频封面提取");
  }
  context.drawImage(video, 0, 0, width, height);

  const blob = await new Promise<Blob>((resolve, reject) => {
    canvas.toBlob((value) => {
      if (!value) {
        reject(new Error("视频封面导出失败"));
        return;
      }
      resolve(value);
    }, "image/jpeg", 0.9);
  });

  URL.revokeObjectURL(objectURL);
  return new File([blob], "cover.jpg", { type: "image/jpeg" });
}
