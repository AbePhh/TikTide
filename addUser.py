from pathlib import Path

import requests


API = "http://127.0.0.1:8080"
VIDEO_DIR = Path(r"D:\code\TikTide\video_test")
PASSWORD = "12345678"
USER_COUNT = 5
VIDEOS_PER_USER = 3
CITIES = ["北京", "上海", "成都", "重庆", "杭州"]


def build_city_tags(video_index: int) -> list[str]:
    first_city = CITIES[video_index % len(CITIES)]
    if video_index % 2 == 0:
        second_city = CITIES[(video_index + 1) % len(CITIES)]
        return [first_city, second_city]
    return [first_city]


def main() -> None:
    if not VIDEO_DIR.exists():
        raise FileNotFoundError(f"Directory not found: {VIDEO_DIR}")

    video_files = sorted(VIDEO_DIR.glob("*.mp4"))
    required_count = USER_COUNT * VIDEOS_PER_USER
    if len(video_files) < required_count:
        raise ValueError(f"Expected at least {required_count} mp4 files in {VIDEO_DIR}")

    for i in range(1, USER_COUNT + 1):
        username = f"testuser{i}"
        user_videos = video_files[(i - 1) * VIDEOS_PER_USER : i * VIDEOS_PER_USER]

        try:
            requests.post(
                f"{API}/api/v1/user/register",
                json={
                    "username": username,
                    "password": PASSWORD,
                },
                timeout=30,
            ).raise_for_status()
        except requests.RequestException:
            pass

        login = requests.post(
            f"{API}/api/v1/user/login",
            json={
                "username": username,
                "password": PASSWORD,
            },
            timeout=30,
        )
        login.raise_for_status()
        token = login.json()["data"]["token"]

        for j, file_path in enumerate(user_videos, start=1):
            global_video_index = (i - 1) * VIDEOS_PER_USER + (j - 1)
            city_tags = build_city_tags(global_video_index)

            cred = requests.post(
                f"{API}/api/v1/video/upload-credential",
                headers={
                    "Authorization": f"Bearer {token}",
                },
                json={
                    "file_name": file_path.name,
                    "content_type": "video/mp4",
                },
                timeout=30,
            )
            cred.raise_for_status()

            cred_data = cred.json()["data"]
            object_key = cred_data["object_key"]
            upload_token = cred_data["upload_token"]

            with file_path.open("rb") as file_obj:
                upload = requests.post(
                    "https://up.qiniup.com/",
                    data={
                        "token": upload_token,
                        "key": object_key,
                    },
                    files={
                        "file": (file_path.name, file_obj, "video/mp4"),
                    },
                    timeout=300,
                )
            upload.raise_for_status()

            publish = requests.post(
                f"{API}/api/v1/video/publish",
                headers={
                    "Authorization": f"Bearer {token}",
                },
                json={
                    "object_key": object_key,
                    "title": f"Test Video {i}-{j}",
                    "hashtag_ids": [],
                    "hashtag_names": city_tags,
                    "allow_comment": 1,
                    "visibility": 1,
                },
                timeout=30,
            )
            publish.raise_for_status()


if __name__ == "__main__":
    main()
