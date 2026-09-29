#!/usr/bin/env python3
"""Inspect a Chromium-generated session PDF for automated Phase 4 evidence.

Usage:
  python tests/inspect_pdf.py <pdf> --min-pages N --contains SUBSTR [--contains SUBSTR...]
       [--min-images N] [--same-page HEADING NEEDLE]
       [--heading-has-drawings HEADING]

Requires pypdf. Optional pymupdf enables --heading-has-drawings.
Exits 0 on success, 1 on failure, 2 if pypdf is missing.
"""

from __future__ import annotations

import argparse
import json
import sys


def count_images(reader) -> int:
    total = 0
    for page in reader.pages:
        try:
            resources = page.get("/Resources")
            if resources is None:
                continue
            resources = resources.get_object()
            xobject = resources.get("/XObject")
            if xobject is None:
                continue
            xobject = xobject.get_object()
            for ref in xobject.values():
                obj = ref.get_object()
                if obj.get("/Subtype") == "/Image":
                    total += 1
        except Exception:
            continue
    return total


def page_texts(reader) -> list[str]:
    return [(reader.pages[i].extract_text() or "") for i in range(len(reader.pages))]


def heading_pages_have_drawings(pdf_path: str, headings: list[str]) -> tuple[dict[str, bool], list[str]]:
    """Return per-heading ok map and problem strings using pymupdf vector drawings."""
    try:
        import fitz
    except ImportError:
        return {}, ["pymupdf is required for --heading-has-drawings"]

    doc = fitz.open(pdf_path)
    ok: dict[str, bool] = {}
    problems: list[str] = []
    for heading in headings:
        found_page = None
        for i in range(doc.page_count):
            page = doc.load_page(i)
            text = page.get_text() or ""
            if heading in text:
                found_page = i
                drawings = page.get_drawings()
                # A stroke polyline becomes several path segments; require a real drawing.
                ok[heading] = len(drawings) >= 1
                if not ok[heading]:
                    problems.append(
                        f"heading {heading!r} on page {i + 1} has no vector drawings"
                    )
                break
        if found_page is None:
            ok[heading] = False
            problems.append(f"heading missing for drawings check: {heading!r}")
    return ok, problems


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("pdf")
    parser.add_argument("--min-pages", type=int, default=2)
    parser.add_argument("--contains", action="append", default=[])
    parser.add_argument("--min-images", type=int, default=0)
    parser.add_argument(
        "--same-page",
        nargs=2,
        action="append",
        default=[],
        metavar=("A", "B"),
        help="Require substrings A and B to appear on the same page",
    )
    parser.add_argument(
        "--heading-has-drawings",
        action="append",
        default=[],
        help="Require the page containing this heading to also have vector drawings",
    )
    parser.add_argument("--json-out", default="")
    args = parser.parse_args()

    try:
        from pypdf import PdfReader
    except ImportError:
        print("pypdf is required for PDF page inspection", file=sys.stderr)
        return 2

    reader = PdfReader(args.pdf)
    pages = len(reader.pages)
    texts = page_texts(reader)
    joined = "\n".join(texts)
    images = count_images(reader)

    problems: list[str] = []
    if pages < args.min_pages:
        problems.append(f"pages={pages} < min {args.min_pages}")
    for needle in args.contains:
        if needle not in joined:
            problems.append(f"missing text: {needle!r}")
    if images < args.min_images:
        problems.append(f"images={images} < min {args.min_images}")
    same_page_ok: dict[str, bool] = {}
    for a, b in args.same_page:
        key = f"{a}||{b}"
        found = any(a in t and b in t for t in texts)
        same_page_ok[key] = found
        if not found:
            problems.append(f"not on same page: {a!r} and {b!r}")

    heading_draw_ok: dict[str, bool] = {}
    if args.heading_has_drawings:
        heading_draw_ok, draw_problems = heading_pages_have_drawings(
            args.pdf, args.heading_has_drawings
        )
        problems.extend(draw_problems)

    report = {
        "pdf": args.pdf,
        "pages": pages,
        "pageChars": [len(t) for t in texts],
        "images": images,
        "contains": {n: (n in joined) for n in args.contains},
        "samePage": same_page_ok,
        "headingHasDrawings": heading_draw_ok,
        "ok": not problems,
        "problems": problems,
    }
    if args.json_out:
        with open(args.json_out, "w", encoding="utf-8") as f:
            json.dump(report, f, ensure_ascii=False, indent=2)
    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 0 if not problems else 1


if __name__ == "__main__":
    sys.exit(main())
