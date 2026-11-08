"""Cache the pinned Windows archive through the GitHub release asset API."""

import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile
from urllib.parse import urlsplit


def main():
    lock = json.loads(Path("tools/bundle-ffmpeg/windows-bundle.json").read_text())
    cache_root = Path(os.environ.get("XDG_CACHE_HOME", Path.home() / ".cache"))
    cache = cache_root / "music-dl" / "media"
    archive = cache / f"windows-media-{lock['sha256']}.tar.gz"

    def verified(path):
        return path.is_file() and hashlib.sha256(path.read_bytes()).hexdigest() == lock["sha256"]

    if verified(archive):
        print("Using verified Windows media archive from cache")
        return

    url = urlsplit(lock["url"])
    parts = url.path.strip("/").split("/")
    if url.scheme != "https" or url.hostname != "github.com" or len(parts) != 6 or parts[2:4] != ["releases", "download"]:
        raise SystemExit("Windows media lock must reference a GitHub release asset")
    repo, tag, name = "/".join(parts[:2]), parts[4], parts[5]
    cache.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(dir=cache, prefix=".download-") as work:
        downloaded = Path(work) / name
        for _ in range(3):
            try:
                result = subprocess.run([
                    "gh", "release", "download", tag, "--repo", repo,
                    "--pattern", name, "--dir", work, "--clobber",
                ], timeout=120)
            except subprocess.TimeoutExpired:
                continue
            if result.returncode == 0:
                if not verified(downloaded):
                    raise SystemExit("Windows media archive checksum mismatch")
                os.replace(downloaded, archive)
                print("Downloaded and verified pinned Windows media archive")
                return
        raise SystemExit("Could not download pinned Windows media archive after three attempts")


if __name__ == "__main__":
    main()
