from __future__ import annotations

import hashlib
import mimetypes
import posixpath
import re
import sys
from dataclasses import dataclass
from pathlib import Path
from urllib.parse import unquote, urlsplit

import markdown
from bs4 import BeautifulSoup
from ebooklib import epub
from pygments.formatters import HtmlFormatter

from read_book import DOCS, ROOT, sync_book_sources


OUTPUT = ROOT / "dist" / "Go服务器开发工程师实战.epub"
PHASE_DIR = re.compile(r"^\d{2}-")
CHAPTER_FILE = re.compile(r"^\d{1,3}-.+\.md$")
BOOK_ID = "https://github.com/dongdonglog/Today_DO_U_GoLang"


@dataclass
class BookStructure:
    intro: Path
    phases: list[tuple[str, Path | None, list[Path]]]
    projects: list[Path]
    ordered: list[Path]


def markdown_converter() -> markdown.Markdown:
    return markdown.Markdown(
        extensions=[
            "tables",
            "admonition",
            "attr_list",
            "md_in_html",
            "pymdownx.details",
            "pymdownx.highlight",
            "pymdownx.inlinehilite",
            "pymdownx.superfences",
            "toc",
        ],
        extension_configs={"toc": {"permalink": True}},
    )


def make_structure() -> BookStructure:
    intro = DOCS / "README.md"
    phases = []
    ordered = [intro]

    for directory in sorted(
        path for path in DOCS.iterdir() if path.is_dir() and PHASE_DIR.match(path.name)
    ):
        stage_intro = directory / "README.md"
        chapters = sorted(
            path
            for path in directory.glob("*.md")
            if CHAPTER_FILE.match(path.name)
        )
        title = directory.name.split("-", 1)[1].replace("-", " ")
        if stage_intro.is_file():
            ordered.append(stage_intro)
        ordered.extend(chapters)
        phases.append((title, stage_intro if stage_intro.is_file() else None, chapters))

    projects_root = DOCS / "毕业项目"
    projects = [projects_root / "README.md"] if (projects_root / "README.md").is_file() else []
    projects.extend(
        sorted(
            path / "README.md"
            for path in projects_root.iterdir()
            if path.is_dir() and (path / "README.md").is_file()
        )
    )
    ordered.extend(projects)
    return BookStructure(intro, phases, projects, ordered)


def resolve_local_path(source: Path, url_path: str) -> Path | None:
    if not url_path:
        return None
    decoded = Path(unquote(url_path))
    candidate = DOCS / decoded.as_posix().lstrip("/") if decoded.is_absolute() else source.parent / decoded
    candidate = candidate.resolve()
    try:
        candidate.relative_to(DOCS.resolve())
    except ValueError:
        return None
    if candidate.is_dir():
        readme = candidate / "README.md"
        return readme if readme.is_file() else candidate
    return candidate


def rendered_soup(source: Path) -> BeautifulSoup:
    converter = markdown_converter()
    return BeautifulSoup(converter.convert(source.read_text(encoding="utf-8")), "html.parser")


def collect_supplemental_docs(ordered: list[Path]) -> list[Path]:
    known = set(ordered)
    supplemental = []
    pending = list(ordered)
    while pending:
        source = pending.pop()
        for anchor in rendered_soup(source).find_all("a", href=True):
            parts = urlsplit(anchor["href"])
            if parts.scheme or parts.netloc or Path(unquote(parts.path)).suffix.lower() not in {".md", ".markdown"}:
                continue
            target = resolve_local_path(source, parts.path)
            if target and target.is_file() and target.suffix.lower() in {".md", ".markdown"} and target not in known:
                known.add(target)
                supplemental.append(target)
                pending.append(target)
    return supplemental


def asset_name(path: Path) -> str:
    relative = path.relative_to(DOCS).as_posix()
    digest = hashlib.sha256(relative.encode("utf-8")).hexdigest()[:16]
    return f"assets/{digest}-{path.name}"


def media_type(path: Path) -> str:
    if path.suffix.lower() == ".svg":
        return "image/svg+xml"
    if path.suffix.lower() in {".go", ".md", ".markdown", ".sql", ".proto", ".yaml", ".yml", ".sh", ".txt", ".mod", ".sum"}:
        return "text/plain"
    return mimetypes.guess_type(path.name)[0] or "application/octet-stream"


def relative_href(from_file: str, to_file: str) -> str:
    return posixpath.relpath(to_file, posixpath.dirname(from_file) or ".")


def extract_title(soup: BeautifulSoup, path: Path) -> str:
    heading = soup.find("h1") or soup.find("h2")
    return heading.get_text(" ", strip=True) if heading else path.stem


def main() -> int:
    sync_book_sources()
    structure = make_structure()
    supplemental = collect_supplemental_docs(structure.ordered)
    docs = structure.ordered + supplemental
    doc_ids = {path: f"chapter-{index:03d}" for index, path in enumerate(docs, start=1)}
    doc_files = {path: f"text/{doc_ids[path]}.xhtml" for path in docs}
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)

    book = epub.EpubBook()
    book.set_identifier(BOOK_ID)
    book.set_title("Go 服务器开发工程师实战：从业务系统到云原生平台")
    book.set_language("zh-CN")
    book.add_metadata(
        "DC",
        "description",
        "76 章 Go 工程实践，包含 SVG 技术图解、可运行示例和毕业项目。",
    )

    style = epub.EpubItem(
        uid="book-style",
        file_name="styles/book.css",
        media_type="text/css",
        content=(
            "body{font-family:serif;line-height:1.65;margin:5%;color:#202729}"
            "h1{font-size:1.65em;line-height:1.3;margin:0 0 1em}"
            "h2{font-size:1.35em;margin:1.8em 0 .65em}"
            "h3{font-size:1.12em;margin:1.4em 0 .5em}"
            "p,li{orphans:2;widows:2}"
            "img,svg{max-width:100%;height:auto}"
            "figure{margin:1.2em 0;text-align:center}"
            "blockquote,.admonition{margin:1em 0;padding:.25em 1em;border-left:3px solid #507d74;background:#f3f6f4}"
            "table{border-collapse:collapse;width:100%;font-size:.88em}"
            "th,td{border:1px solid #aebbb6;padding:.4em;vertical-align:top}"
            "pre{white-space:pre-wrap;overflow-wrap:anywhere;padding:.8em;background:#f1f3f2;border:1px solid #d8dfdc;font-size:.78em;line-height:1.45}"
            "code{font-family:monospace;font-size:.9em;overflow-wrap:anywhere}"
            ".highlight{margin:1em 0}"
            "hr{border:0;border-top:1px solid #bfc9c5;margin:2em 0}"
            "a{color:#175f78;text-decoration:underline}"
            + HtmlFormatter(style="friendly").get_style_defs(".highlight")
        ).encode("utf-8"),
    )
    book.add_item(style)

    assets: dict[Path, epub.EpubItem] = {}

    def add_asset(path: Path) -> epub.EpubItem:
        path = path.resolve()
        if path not in assets:
            digest = hashlib.sha256(path.relative_to(DOCS).as_posix().encode("utf-8")).hexdigest()[:16]
            item = epub.EpubItem(
                uid=f"asset-{digest}",
                file_name=asset_name(path),
                media_type=media_type(path),
                content=path.read_bytes(),
            )
            assets[path] = item
            book.add_item(item)
        return assets[path]

    rendered_docs: dict[Path, epub.EpubHtml] = {}
    titles: dict[Path, str] = {}
    for path in docs:
        soup = rendered_soup(path)
        title = extract_title(soup, path)
        titles[path] = title
        chapter = epub.EpubHtml(
            uid=doc_ids[path],
            title=title,
            file_name=doc_files[path],
            lang="zh-CN",
        )
        chapter.add_link(href="../styles/book.css", rel="stylesheet", type="text/css")

        for image in soup.find_all("img", src=True):
            parts = urlsplit(image["src"])
            target = resolve_local_path(path, parts.path)
            if not parts.scheme and not parts.netloc and target and target.is_file():
                item = add_asset(target)
                image["src"] = relative_href(chapter.file_name, item.file_name)
                if parts.fragment:
                    image["src"] += f"#{parts.fragment}"

        for anchor in soup.find_all("a", href=True):
            original = anchor["href"]
            parts = urlsplit(original)
            if parts.scheme or parts.netloc or not parts.path:
                continue
            target = resolve_local_path(path, parts.path)
            if not target or not target.is_file():
                continue
            if target in doc_files:
                destination = doc_files[target]
            elif target.suffix.lower() in {".md", ".markdown"}:
                continue
            else:
                destination = add_asset(target).file_name
            anchor["href"] = relative_href(chapter.file_name, destination)
            if parts.fragment:
                anchor["href"] += f"#{parts.fragment}"
            if parts.query:
                anchor["href"] += f"?{parts.query}"

        chapter.content = str(soup).encode("utf-8")
        book.add_item(chapter)
        rendered_docs[path] = chapter

    book.add_item(epub.EpubNcx())
    book.add_item(epub.EpubNav())

    toc = [epub.Link(doc_files[structure.intro], "阅读指南", doc_ids[structure.intro])]
    spine = ["nav", rendered_docs[structure.intro]]
    for phase_title, stage_intro, chapters in structure.phases:
        children = []
        if stage_intro:
            item = rendered_docs[stage_intro]
            children.append(epub.Link(item.file_name, titles[stage_intro], item.id))
            spine.append(item)
        for path in chapters:
            item = rendered_docs[path]
            children.append(epub.Link(item.file_name, titles[path], item.id))
            spine.append(item)
        toc.append((epub.Section(phase_title), tuple(children)))

    if structure.projects:
        children = []
        for path in structure.projects:
            item = rendered_docs[path]
            children.append(epub.Link(item.file_name, titles[path], item.id))
            spine.append(item)
        toc.append((epub.Section("毕业项目"), tuple(children)))

    if supplemental:
        children = []
        for path in supplemental:
            item = rendered_docs[path]
            children.append(epub.Link(item.file_name, titles[path], item.id))
            spine.append(item)
        toc.append((epub.Section("示例项目说明"), tuple(children)))

    book.toc = tuple(toc)
    book.spine = spine
    epub.write_epub(str(OUTPUT), book)
    print(f"已生成：{OUTPUT}")
    print(f"正文文档：{len(docs)}，本地配图和示例资源：{len(assets)}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ImportError as error:
        print(
            "缺少 EPUB 构建依赖，请运行：uv run --with-requirements requirements-book.txt python scripts/build_epub.py",
            file=sys.stderr,
        )
        raise SystemExit(1) from error
