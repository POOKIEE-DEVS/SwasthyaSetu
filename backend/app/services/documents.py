"""Checks on uploaded verification documents.

The file type is decided from the file's first bytes, never from the name
or the browser's Content-Type, so a script renamed to .jpg is refused and
documents are always served back with a type that can't run code.
"""

from __future__ import annotations

from dataclasses import dataclass

from fastapi import UploadFile

IMAGE_TYPES = ("image/jpeg", "image/png", "image/webp")
PDF_TYPE = "application/pdf"


class DocumentError(Exception):
    """A document was refused. The message is safe to show to the user."""


@dataclass
class CheckedDocument:
    kind: str
    content_type: str
    data: bytes


def sniff_type(data: bytes) -> str | None:
    if data.startswith(b"\xff\xd8\xff"):
        return "image/jpeg"
    if data.startswith(b"\x89PNG\r\n\x1a\n"):
        return "image/png"
    if data[:4] == b"RIFF" and data[8:12] == b"WEBP":
        return "image/webp"
    if data.startswith(b"%PDF-"):
        return PDF_TYPE
    return None


def is_provided(upload: UploadFile | None) -> bool:
    return upload is not None and bool(upload.filename)


def check_upload(
    upload: UploadFile,
    *,
    kind: str,
    label: str,
    max_bytes: int,
    allow_pdf: bool = True,
) -> CheckedDocument:
    data = upload.file.read(max_bytes + 1)
    if not data:
        raise DocumentError(f"{label}: the file is empty.")
    if len(data) > max_bytes:
        raise DocumentError(
            f"{label}: the file is too large (max {max_bytes // (1024 * 1024)} MB)."
        )
    content_type = sniff_type(data)
    allowed = IMAGE_TYPES + ((PDF_TYPE,) if allow_pdf else ())
    if content_type not in allowed:
        kinds = "a JPEG, PNG or WebP photo" + (" or a PDF" if allow_pdf else "")
        raise DocumentError(f"{label}: please upload {kinds}.")
    return CheckedDocument(kind=kind, content_type=content_type, data=data)
