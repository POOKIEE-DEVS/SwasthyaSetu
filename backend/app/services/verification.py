"""Professional verification: applications, their documents, and reviews.

Rules
- One application per account. A rejected applicant fixes the problem and
  resubmits the same application, which goes back to "pending".
- Approved is final for the applicant (they can't resubmit over it). The
  admin can still revoke it by rejecting.
- Someone is a *verified professional* only while their application is
  approved and their account role matches the application's role.
- Every submission and decision is written to the audit log.
"""

from __future__ import annotations

from sqlmodel import Session, col, select

from app.models import Application, AuditEvent, Document, User, timestamp, utcnow
from app.schemas.verification import (
    AdminApplication,
    ApplicationPublic,
    AuditPublic,
    DocumentInfo,
    VerificationSummary,
)
from app.services.documents import CheckedDocument


def application_for(db: Session, user: User) -> Application | None:
    return db.exec(select(Application).where(Application.user_id == user.id)).first()


def is_verified(user: User, application: Application | None) -> bool:
    return (
        application is not None
        and application.status == "approved"
        and user.role == application.role
    )


def _documents(db: Session, application: Application) -> list[DocumentInfo]:
    rows = db.exec(
        select(Document.id, Document.kind, Document.content_type, Document.size)
        .where(Document.application_id == application.id)
        .order_by(col(Document.id))
    ).all()
    return [
        DocumentInfo(id=i, kind=kind, content_type=ctype, size=size)
        for i, kind, ctype, size in rows
    ]


def to_public(db: Session, application: Application) -> ApplicationPublic:
    assert application.id is not None
    return ApplicationPublic(
        id=application.id,
        role=application.role,
        full_name=application.full_name,
        phone=application.phone,
        citizenship_number=application.citizenship_number,
        citizenship_district=application.citizenship_district,
        council_number=application.council_number,
        institution=application.institution,
        recommender_name=application.recommender_name,
        recommender_nmc=application.recommender_nmc,
        status=application.status,
        rejection_reason=application.rejection_reason,
        submitted_at=timestamp(application.submitted_at),
        reviewed_at=timestamp(application.reviewed_at),
        documents=_documents(db, application),
    )


def to_admin(db: Session, application: Application) -> AdminApplication:
    user = db.get(User, application.user_id)
    assert user is not None
    history = db.exec(
        select(AuditEvent)
        .where(AuditEvent.application_id == application.id)
        .order_by(col(AuditEvent.id))
    ).all()
    return AdminApplication(
        **to_public(db, application).model_dump(),
        user_email=user.email,
        user_name=user.name,
        user_picture_url=user.picture_url,
        history=[
            AuditPublic(
                action=e.action,
                actor_email=e.actor_email,
                reason=e.reason,
                at=timestamp(e.at),
            )
            for e in history
        ],
    )


def summary(application: Application | None) -> VerificationSummary | None:
    if application is None:
        return None
    return VerificationSummary(
        role=application.role,
        status=application.status,
        rejection_reason=application.rejection_reason,
        full_name=application.full_name,
    )


def submit(
    db: Session,
    user: User,
    fields: dict[str, str | None],
    documents: list[CheckedDocument],
) -> Application:
    """Create the application, or replace a pending/rejected one."""
    application = application_for(db, user)
    action = "submitted"
    if application is None:
        application = Application(user_id=user.id, **fields)
    else:
        action = "resubmitted"
        for name, value in fields.items():
            setattr(application, name, value)
        application.status = "pending"
        application.rejection_reason = None
        application.reviewed_at = None
        application.reviewed_by = None
        application.submitted_at = utcnow()
        for old in db.exec(
            select(Document).where(Document.application_id == application.id)
        ).all():
            db.delete(old)

    # Applying as a professional makes that the account's role.
    user.role = fields["role"]
    db.add(user)
    db.add(application)
    db.flush()
    for doc in documents:
        db.add(
            Document(
                application_id=application.id,
                kind=doc.kind,
                content_type=doc.content_type,
                size=len(doc.data),
                data=doc.data,
            )
        )
    db.add(
        AuditEvent(application_id=application.id, actor_email=user.email, action=action)
    )
    db.commit()
    db.refresh(application)
    return application


def decide(
    db: Session,
    application: Application,
    admin: User,
    *,
    approve: bool,
    reason: str | None = None,
) -> Application:
    application.status = "approved" if approve else "rejected"
    application.rejection_reason = None if approve else reason
    application.reviewed_at = utcnow()
    application.reviewed_by = admin.email
    db.add(application)
    db.add(
        AuditEvent(
            application_id=application.id,
            actor_email=admin.email,
            action="approved" if approve else "rejected",
            reason=reason,
        )
    )
    db.commit()
    db.refresh(application)
    return application
