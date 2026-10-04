"""Each professional's record of the people they have helped.

A call is recorded once, when an accepted call ends, with the patient's
name, when it started (the moment the professional accepted) and how long
it lasted. Only the professional who took the call can read it back.

Recording never gets in the way of the call: if the database is down, the
call still ends normally and the record is skipped (and logged).
"""

from __future__ import annotations

import logging
import time
from datetime import UTC, datetime

from sqlalchemy.exc import IntegrityError
from sqlmodel import Session, col, func, select

from app.db import database_ready, get_engine
from app.models import HelpRecord, timestamp
from app.schemas.consultation import HelpedPerson, HelpSummary
from app.services.consultations import Consultation

logger = logging.getLogger(__name__)

# The list a professional sees; the count always covers every call.
RECENT_LIMIT = 50


def record_call(consultation: Consultation, ended_at: float | None = None) -> bool:
    """Saves one helped patient for the professional who took the call.

    Blocking (database I/O): call it through run_in_threadpool. Returns
    whether a record was written; never raises.
    """
    if consultation.professional_user_id is None or consultation.accepted_at is None:
        return False
    if not database_ready():
        logger.warning("help record skipped: database unavailable")
        return False
    ended = ended_at if ended_at is not None else time.time()
    record = HelpRecord(
        professional_id=consultation.professional_user_id,
        consultation_id=consultation.id,
        patient_name=consultation.patient_name,
        started_at=datetime.fromtimestamp(consultation.accepted_at, UTC),
        ended_at=datetime.fromtimestamp(ended, UTC),
        duration_seconds=max(0, round(ended - consultation.accepted_at)),
    )
    try:
        with Session(get_engine()) as db:
            db.add(record)
            db.commit()
    except IntegrityError:
        # Already recorded (both participants report the end).
        return False
    except Exception as exc:
        logger.warning("could not save help record", exc_info=exc)
        return False
    return True


def summary_for(db: Session, professional_id: int) -> HelpSummary:
    count = db.exec(
        select(func.count())
        .select_from(HelpRecord)
        .where(HelpRecord.professional_id == professional_id)
    ).one()
    rows = db.exec(
        select(HelpRecord)
        .where(HelpRecord.professional_id == professional_id)
        .order_by(col(HelpRecord.started_at).desc(), col(HelpRecord.id).desc())
        .limit(RECENT_LIMIT)
    ).all()
    return HelpSummary(
        count=count,
        people=[
            HelpedPerson(
                patient_name=row.patient_name,
                started_at=timestamp(row.started_at) or 0.0,
                duration_seconds=row.duration_seconds,
            )
            for row in rows
        ],
    )
