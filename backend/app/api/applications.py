"""A medical professional applies for verification (KYC).

Everyone uploads both sides of their citizenship certificate. Then:
- Doctor: Nepal Medical Council (NMC) number + NMC certificate.
- Pharmacist: Nepal Pharmacy Council number + its certificate.
- Nurse: Nepal Nursing Council number + its certificate.
- Paramedic (health assistant, CMA and similar): Nepal Health
  Professional Council number + its certificate.
- MBBS student: college, the recommending doctor's name and NMC number,
  and the recommendation letter.
A selfie holding the citizenship certificate is optional.

The admin reviews every application by hand (see admin.py). Until it is
approved, the applicant can't see or call patients.
"""

from __future__ import annotations

import re
from typing import Annotated

from fastapi import APIRouter, Depends, File, Form, HTTPException, UploadFile, status
from sqlmodel import Session

from app.api.deps import current_user
from app.core.config import settings
from app.db import get_session
from app.models import User
from app.schemas.verification import ApplicationPublic, ProfessionalRole
from app.services import verification
from app.services.documents import (
    CheckedDocument,
    DocumentError,
    check_upload,
    is_provided,
)

router = APIRouter(prefix="/applications", tags=["verification"])

# The council each registered profession is checked against.
COUNCILS: dict[str, str] = {
    "doctor": "Nepal Medical Council",
    "pharmacist": "Nepal Pharmacy Council",
    "nurse": "Nepal Nursing Council",
    "paramedic": "Nepal Health Professional Council",
}

PHONE_RE = re.compile(r"^\+?[0-9][0-9 -]{6,18}$")
ID_NUMBER_RE = re.compile(r"^[0-9A-Za-z][0-9A-Za-z /-]{0,39}$")

Text = Annotated[str, Form(max_length=160)]
OptionalText = Annotated[str | None, Form(max_length=160)]
OptionalFile = Annotated[UploadFile | None, File()]


def _clean(value: str | None) -> str | None:
    value = (value or "").strip()
    return value or None


def _bad(detail: str) -> HTTPException:
    return HTTPException(status.HTTP_422_UNPROCESSABLE_CONTENT, detail)


@router.get("/me", response_model=ApplicationPublic | None)
def my_application(
    user: User = Depends(current_user), db: Session = Depends(get_session)
) -> ApplicationPublic | None:
    application = verification.application_for(db, user)
    return verification.to_public(db, application) if application else None


@router.post("", response_model=ApplicationPublic, status_code=status.HTTP_201_CREATED)
def submit_application(
    role: Annotated[ProfessionalRole, Form()],
    # Defaults instead of "required", so a blank field gets a readable
    # message below rather than FastAPI's generic validation list.
    full_name: Text = "",
    phone: Text = "",
    citizenship_number: Text = "",
    citizenship_district: Text = "",
    consent: Annotated[bool, Form()] = False,
    citizenship_front: OptionalFile = None,
    citizenship_back: OptionalFile = None,
    council_number: OptionalText = None,
    institution: OptionalText = None,
    recommender_name: OptionalText = None,
    recommender_nmc: OptionalText = None,
    council_certificate: OptionalFile = None,
    recommendation_letter: OptionalFile = None,
    selfie: OptionalFile = None,
    user: User = Depends(current_user),
    db: Session = Depends(get_session),
) -> ApplicationPublic:
    if not consent:
        raise _bad("Please agree to the document check before submitting.")
    existing = verification.application_for(db, user)
    if existing is not None and existing.status == "approved":
        raise HTTPException(status.HTTP_409_CONFLICT, "You are already verified.")

    fields: dict[str, str | None] = {
        "role": role,
        "full_name": _clean(full_name),
        "phone": _clean(phone),
        "citizenship_number": _clean(citizenship_number),
        "citizenship_district": _clean(citizenship_district),
        "council_number": None,
        "institution": None,
        "recommender_name": None,
        "recommender_nmc": None,
    }
    if not fields["full_name"] or len(fields["full_name"]) < 2:
        raise _bad("Please enter your full name as on your citizenship.")
    if not PHONE_RE.match(fields["phone"] or ""):
        raise _bad("Please enter a valid phone number.")
    if not ID_NUMBER_RE.match(fields["citizenship_number"] or ""):
        raise _bad("Please enter your citizenship number.")
    if not fields["citizenship_district"]:
        raise _bad("Please enter the district that issued your citizenship.")

    if not is_provided(citizenship_front) or not is_provided(citizenship_back):
        raise _bad("Please upload both sides of your citizenship certificate.")
    assert citizenship_front is not None and citizenship_back is not None

    max_bytes = settings.max_upload_bytes
    try:
        documents: list[CheckedDocument] = [
            check_upload(
                citizenship_front,
                kind="citizenship_front",
                label="Citizenship (front)",
                max_bytes=max_bytes,
            ),
            check_upload(
                citizenship_back,
                kind="citizenship_back",
                label="Citizenship (back)",
                max_bytes=max_bytes,
            ),
        ]

        if role in COUNCILS:
            council = COUNCILS[role]
            number = _clean(council_number)
            if not number or not ID_NUMBER_RE.match(number):
                raise _bad(f"Please enter your {council} registration number.")
            fields["council_number"] = number
            if not is_provided(council_certificate):
                raise _bad(f"Please upload your {council} certificate.")
            assert council_certificate is not None
            documents.append(
                check_upload(
                    council_certificate,
                    kind="council_certificate",
                    label=f"{council} certificate",
                    max_bytes=max_bytes,
                )
            )
        else:  # MBBS student
            fields["institution"] = _clean(institution)
            fields["recommender_name"] = _clean(recommender_name)
            fields["recommender_nmc"] = _clean(recommender_nmc)
            if not fields["institution"]:
                raise _bad("Please enter your medical college.")
            if not fields["recommender_name"]:
                raise _bad("Please enter the name of the doctor recommending you.")
            if not ID_NUMBER_RE.match(fields["recommender_nmc"] or ""):
                raise _bad("Please enter the recommending doctor's NMC number.")
            if not is_provided(recommendation_letter):
                raise _bad("Please upload the doctor's letter of recommendation.")
            assert recommendation_letter is not None
            documents.append(
                check_upload(
                    recommendation_letter,
                    kind="recommendation_letter",
                    label="Letter of recommendation",
                    max_bytes=max_bytes,
                )
            )

        if is_provided(selfie):
            assert selfie is not None
            documents.append(
                check_upload(
                    selfie,
                    kind="selfie",
                    label="Selfie with citizenship",
                    max_bytes=max_bytes,
                    allow_pdf=False,
                )
            )
    except DocumentError as exc:
        raise _bad(str(exc)) from exc

    application = verification.submit(db, user, fields, documents)
    return verification.to_public(db, application)
