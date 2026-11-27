import contextlib
import io
import tempfile
import unittest
from pathlib import Path

from scripts.changelog_release_notes import extract_release_notes, main


CHANGELOG = """# Changelog

## [Unreleased]

### Added

- Work in progress.

## [1.2.3] - 2026-11-27

### Added

- Automatic update checks.

### Fixed

- Radio playback after pausing a podcast.

## [1.2.2] - 2026-11-20

### Changed

- Improved navigation.
"""


class ExtractReleaseNotesTests(unittest.TestCase):
    def test_extracts_matching_version(self):
        notes = extract_release_notes(CHANGELOG, "v1.2.3")
        self.assertIn("Automatic update checks.", notes)
        self.assertIn("Radio playback after pausing a podcast.", notes)
        self.assertNotIn("Improved navigation.", notes)
        self.assertNotIn("## [1.2.3]", notes)

    def test_accepts_tag_without_v_prefix(self):
        self.assertEqual(
            extract_release_notes(CHANGELOG, "1.2.3"),
            extract_release_notes(CHANGELOG, "v1.2.3"),
        )

    def test_missing_version_fails(self):
        with self.assertRaisesRegex(ValueError, "no changelog section"):
            extract_release_notes(CHANGELOG, "v9.9.9")

    def test_version_match_is_exact(self):
        with self.assertRaisesRegex(ValueError, "no changelog section"):
            extract_release_notes(CHANGELOG, "v1.2.30")

    def test_empty_version_fails(self):
        with self.assertRaisesRegex(ValueError, "empty"):
            extract_release_notes(CHANGELOG, "v")

    def test_empty_section_fails(self):
        changelog = "## [1.2.3] - 2026-11-27\n\n## [1.2.2] - 2026-11-20\n\n- Old.\n"
        with self.assertRaisesRegex(ValueError, "empty"):
            extract_release_notes(changelog, "v1.2.3")

    def test_duplicate_sections_fail(self):
        changelog = "## [1.2.3]\n\n- One.\n\n## [1.2.3]\n\n- Two.\n"
        with self.assertRaisesRegex(ValueError, "multiple changelog sections"):
            extract_release_notes(changelog, "v1.2.3")

    def test_extracts_last_version_through_end_of_file(self):
        notes = extract_release_notes(CHANGELOG, "v1.2.2")
        self.assertEqual(notes, "### Changed\n\n- Improved navigation.\n")

    def test_cli_writes_release_notes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            changelog = root / "CHANGELOG.md"
            output = root / "release-notes.md"
            changelog.write_text(CHANGELOG, encoding="utf-8")

            result = main(["v1.2.3", str(changelog), str(output)])

            self.assertEqual(result, 0)
            self.assertIn("Automatic update checks.", output.read_text(encoding="utf-8"))

    def test_cli_reports_missing_version(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            changelog = root / "CHANGELOG.md"
            output = root / "release-notes.md"
            changelog.write_text(CHANGELOG, encoding="utf-8")

            stderr = io.StringIO()
            with contextlib.redirect_stderr(stderr):
                result = main(["v9.9.9", str(changelog), str(output)])

            self.assertEqual(result, 1)
            self.assertIn("no changelog section found", stderr.getvalue())
            self.assertFalse(output.exists())


if __name__ == "__main__":
    unittest.main()
