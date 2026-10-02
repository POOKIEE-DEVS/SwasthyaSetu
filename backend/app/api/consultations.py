"""Patient requests a doctor; a verified professional accepts; either ends.

Patients never need to sign in: an emergency must not wait on a login.
Seeing the waiting list (which includes shared chats) and accepting a
patient are for verified doctors, pharmacists and MBBS students only.
"""

from __future__ import annotations

from fastapi import APIRouter, Depends, HTTPException, Response, status

from app.api.deps import VerifiedProfessional, require_professional
from app.realtime.ice import get_ice_servers
from app.realtime.websocket import broadcast_queue
from app.schemas.consultation import (
    CallTicket,
    ConsultationCreate,
    ConsultationPublic,
    EndCall,
)
from app.services import consultations
from app.services.consultations import (
    ConsultationNotFoundError,
    ConsultationUnavailableError,
)

router = APIRouter(prefix="/consultations", tags=["consultations"])


@router.post("", response_model=CallTicket, status_code=status.HTTP_201_CREATED)
async def request_doctor(body: ConsultationCreate) -> CallTicket:
    consultation = consultations.create(body.patient_name, body.summary)
    await broadcast_queue()
    return CallTicket(
        consultation=consultation.public(),
        role="patient",
        token=consultation.patient_token,
        ice_servers=await get_ice_servers(),
    )


@router.get("", response_model=list[ConsultationPublic])
async def waiting_patients(
    _pro: VerifiedProfessional = Depends(require_professional),
) -> list[ConsultationPublic]:
    return [c.public() for c in consultations.waiting()]


@router.post("/{consultation_id}/accept", response_model=CallTicket)
async def accept(
    consultation_id: str,
    pro: VerifiedProfessional = Depends(require_professional),
) -> CallTicket:
    try:
        consultation = consultations.accept(consultation_id, pro.badge)
    except ConsultationNotFoundError as exc:
        raise HTTPException(
            status.HTTP_404_NOT_FOUND, "This request no longer exists."
        ) from exc
    except ConsultationUnavailableError as exc:
        raise HTTPException(
            status.HTTP_409_CONFLICT, "Another doctor has already taken this request."
        ) from exc

    await broadcast_queue()
    assert consultation.doctor_token is not None
    return CallTicket(
        consultation=consultation.public(),
        role="doctor",
        token=consultation.doctor_token,
        ice_servers=await get_ice_servers(),
    )


@router.post(
    "/{consultation_id}/end",
    status_code=status.HTTP_204_NO_CONTENT,
    response_class=Response,
)
async def end(consultation_id: str, body: EndCall) -> Response:
    try:
        consultations.end(consultation_id, body.token)
    except ConsultationNotFoundError as exc:
        raise HTTPException(status.HTTP_404_NOT_FOUND, "Not found.") from exc
    # A patient who gives up while still waiting leaves the doctors' queue.
    await broadcast_queue()
    return Response(status_code=status.HTTP_204_NO_CONTENT)
