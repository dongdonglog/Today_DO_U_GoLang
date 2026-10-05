from __future__ import annotations

import re
import shutil
import subprocess
import sys
import threading
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / ".book-docs"
PHASE_DIR = re.compile(r"^\d{2}-")
IGNORE = shutil.ignore_patterns(
    ".DS_Store",
    ".git",
    ".venv",
    "__pycache__",
    "images/src",
    "site",
)
WATCH_IGNORED_PARTS = {".DS_Store", ".git", ".venv", "__pycache__", "site"}


def sync_book_sources() -> None:
    DOCS.mkdir(exist_ok=True)
    for name in ("README.md", "outline.md"):
        shutil.copy2(ROOT / name, DOCS / name)

    for source in ROOT.iterdir():
        if not source.is_dir() or not (
            PHASE_DIR.match(source.name) or source.name == "毕业项目"
        ):
            continue
        shutil.copytree(
            source,
            DOCS / source.name,
            dirs_exist_ok=True,
            ignore=IGNORE,
        )

    for name in ("images", "stylesheets"):
        source = ROOT / name
        if source.is_dir():
            shutil.copytree(source, DOCS / name, dirs_exist_ok=True, ignore=IGNORE)


def source_signature() -> tuple[tuple[str, int, int], ...]:
    files = [ROOT / "README.md", ROOT / "outline.md"]
    sources = [ROOT / "images", ROOT / "stylesheets"]
    for source in ROOT.iterdir():
        if source.is_dir() and (
            PHASE_DIR.match(source.name) or source.name == "毕业项目"
        ):
            sources.append(source)

    for source in sources:
        if not source.is_dir():
            continue
        for path in source.rglob("*"):
            relative = path.relative_to(source)
            if any(part in WATCH_IGNORED_PARTS for part in relative.parts):
                continue
            if any(
                relative.parts[index : index + 2] == ("images", "src")
                for index in range(len(relative.parts) - 1)
            ):
                continue
            if path.is_file():
                files.append(path)

    signature = []
    for path in files:
        try:
            stat = path.stat()
        except FileNotFoundError:
            continue
        signature.append(
            (path.relative_to(ROOT).as_posix(), stat.st_mtime_ns, stat.st_size)
        )
    return tuple(sorted(signature))


def watch_sources(stop: threading.Event) -> None:
    previous = source_signature()
    while not stop.wait(1):
        current = source_signature()
        if current == previous:
            continue
        if stop.wait(0.4):
            return
        sync_book_sources()
        previous = source_signature()
        print("书稿已同步到阅读站。", flush=True)


if __name__ == "__main__":
    sync_book_sources()
    stop_watching = threading.Event()
    watcher = threading.Thread(
        target=watch_sources,
        args=(stop_watching,),
        daemon=True,
    )
    watcher.start()
    try:
        exit_code = subprocess.call(
            [sys.executable, "-m", "mkdocs", "serve", *sys.argv[1:]], cwd=ROOT
        )
    except KeyboardInterrupt:
        exit_code = 0
    finally:
        stop_watching.set()
        watcher.join(timeout=2)
    raise SystemExit(exit_code)
