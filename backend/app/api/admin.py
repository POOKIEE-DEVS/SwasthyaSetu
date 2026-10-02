"""Admin review of professional applications.

One admin (any Google account in ADMIN_EMAILS) checks each applicant's
documents by hand, looks the council number up on the council's register,
then approves or rejects with a reason the applicant will see.
"""

from __future__ import annotations

from typing import Literal

from fastapi import APIRouter, Depends, HTTPException, Query, Response, status
from sqlmodel import Session, col, select

from app.api.deps import require_admin
from app.db import get_session
from app.models import Application, Document, User
from app.schemas.verification import AdminApplication, RejectRequest
from app.services import verification

router = APIRouter(prefix="/admin", tags=["admin"])


def _get_application(db: Session, application_id: int) -> Application:
    application = db.get(Application, application_id)
    if application is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Application not found.")
    return application


@router.get("/applications", response_model=list[AdminApplication])
def list_applications(
    status_filter: Literal["pending", "approved", "rejected", "all"] = Query(
        "pending", alias="status"
    ),
    _admin: User = Depends(require_admin),
    db: Session = Depends(get_session),
) -> list[AdminApplication]:
    query = select(Application)
    if status_filter != "all":
        query = query.where(Application.status == status_filter)
    # Oldest first while waiting (fair queue); newest first otherwise.
    order = col(Application.submitted_at)
    query = query.order_by(order if status_filter == "pending" else order.desc())
    return [verification.to_admin(db, a) for a in db.exec(query).all()]


@router.get("/documents/{document_id}")
def get_document(
    document_id: int,
    _admin: User = Depends(require_admin),
    db: Session = Depends(get_session),
) -> Response:
    document = db.get(Document, document_id)
    if document is None:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Document not found.")
    return Response(
        content=document.data,
        # Detected from the bytes at upload time: only images and PDFs.
        media_type=document.content_type,
        headers={
            "Content-Disposition": "inline",
            "Cache-Control": "private, no-store",
            "X-Content-Type-Options": "nosniff",
        },
    )


@router.post("/applications/{application_id}/approve", response_model=AdminApplication)
def approve(
    application_id: int,
    admin: User = Depends(require_admin),
    db: Session = Depends(get_session),
) -> AdminApplication:
    application = _get_application(db, application_id)
    if application.status == "approved":
        raise HTTPException(status.HTTP_409_CONFLICT, "Already approved.")
    verification.decide(db, application, admin, approve=True)
    return verification.to_admin(db, application)


@router.post("/applications/{application_id}/reject", response_model=AdminApplication)
def reject(
    application_id: int,
    body: RejectRequest,
    admin: User = Depends(require_admin),
    db: Session = Depends(get_session),
) -> AdminApplication:
    """Also revokes an approval (e.g. a licence turns out to be invalid)."""
    application = _get_application(db, application_id)
    if application.status == "rejected":
        raise HTTPException(status.HTTP_409_CONFLICT, "Already rejected.")
    verification.decide(
        db, application, admin, approve=False, reason=body.reason.strip()
    )
    return verification.to_admin(db, application)
