#!/usr/bin/env python3
"""Extract a version's release notes from a Keep a Changelog-style file."""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path


VERSION_HEADING = re.compile(r"^##\s+\[([^\]]+)\](?:\s+-\s+.*)?\s*$")


def extract_release_notes(changelog: str, tag: str) -> str:
    """Return the Markdown body for *tag*, without its version heading."""
    version = tag[1:] if tag.startswith("v") else tag
    if not version:
        raise ValueError("release tag is empty")

    lines = changelog.splitlines()
    matches = [i for i, line in enumerate(lines) if heading_version(line) == version]
    if not matches:
        raise ValueError(f"no changelog section found for {tag}")
    if len(matches) > 1:
        raise ValueError(f"multiple changelog sections found for {tag}")

    start = matches[0] + 1
    end = len(lines)
    for i in range(start, len(lines)):
        if VERSION_HEADING.match(lines[i]):
            end = i
            break

    notes = "\n".join(lines[start:end]).strip()
    if not notes:
        raise ValueError(f"changelog section for {tag} is empty")
    return notes + "\n"


def heading_version(line: str) -> str | None:
    match = VERSION_HEADING.match(line)
    return match.group(1) if match else None


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("tag", help="release tag, for example v1.2.3")
    parser.add_argument("changelog", type=Path, help="path to CHANGELOG.md")
    parser.add_argument("output", type=Path, help="path for extracted release notes")
    args = parser.parse_args(argv)

    try:
        changelog = args.changelog.read_text(encoding="utf-8")
        notes = extract_release_notes(changelog, args.tag)
        args.output.write_text(notes, encoding="utf-8")
    except (OSError, UnicodeError, ValueError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
