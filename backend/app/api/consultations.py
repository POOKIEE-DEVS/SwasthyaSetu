"""Patient requests a doctor; a doctor accepts; either ends the call.

The doctor side has no login for the demo, so anyone who opens the doctor
page can accept. That is fine for a staged demo, and it is the first thing
to change before real use.
"""

from __future__ import annotations

from fastapi import APIRouter, HTTPException, Response, status

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
async def waiting_patients() -> list[ConsultationPublic]:
    return [c.public() for c in consultations.waiting()]


@router.post("/{consultation_id}/accept", response_model=CallTicket)
async def accept(consultation_id: str) -> CallTicket:
    try:
        consultation = consultations.accept(consultation_id)
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
