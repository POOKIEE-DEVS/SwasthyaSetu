from __future__ import annotations

import pytest
from fastapi.testclient import TestClient
from starlette.websockets import WebSocketDisconnect

from app.realtime.websocket import CLOSE_REPLACED, CLOSE_UNAUTHORIZED


def start_call(client: TestClient) -> tuple[str, str, str]:
    patient = client.post("/api/v1/consultations", json={"patient_name": "Ram"}).json()
    cid = patient["consultation"]["id"]
    doctor = client.post(f"/api/v1/consultations/{cid}/accept").json()
    return cid, patient["token"], doctor["token"]


def test_doctor_queue_snapshot_and_updates(client: TestClient) -> None:
    with client.websocket_connect("/ws/doctors") as queue:
        assert queue.receive_json() == {"type": "queue", "consultations": []}

        client.post("/api/v1/consultations", json={"patient_name": "Gita"})

        update = queue.receive_json()
        assert update["type"] == "queue"
        assert [c["patient_name"] for c in update["consultations"]] == ["Gita"]


def test_bad_token_is_rejected(client: TestClient) -> None:
    cid, _, _ = start_call(client)
    with (
        pytest.raises(WebSocketDisconnect) as exc,
        client.websocket_connect(f"/ws/consultations/{cid}?token=wrong"),
    ):
        pass
    assert exc.value.code == CLOSE_UNAUTHORIZED


def test_unknown_consultation_is_rejected(client: TestClient) -> None:
    with (
        pytest.raises(WebSocketDisconnect) as exc,
        client.websocket_connect("/ws/consultations/missing?token=x"),
    ):
        pass
    assert exc.value.code == CLOSE_UNAUTHORIZED


def test_signals_relay_between_patient_and_doctor(client: TestClient) -> None:
    cid, patient_token, doctor_token = start_call(client)

    with client.websocket_connect(
        f"/ws/consultations/{cid}?token={patient_token}"
    ) as p:
        joined = p.receive_json()
        assert joined == {"type": "joined", "role": "patient", "peer_present": False}

        with client.websocket_connect(
            f"/ws/consultations/{cid}?token={doctor_token}"
        ) as d:
            assert d.receive_json()["peer_present"] is True
            # The one already in the room is told, and makes the offer.
            assert p.receive_json() == {"type": "peer-joined", "role": "doctor"}

            p.send_json({"type": "offer", "payload": {"sdp": "v=0", "type": "offer"}})
            offer = d.receive_json()
            assert offer["type"] == "offer"
            assert offer["from"] == "patient"
            assert offer["payload"]["sdp"] == "v=0"

            d.send_json({"type": "answer", "payload": {"sdp": "v=0", "type": "answer"}})
            assert p.receive_json()["type"] == "answer"

        assert p.receive_json() == {"type": "peer-left", "role": "doctor"}


def test_unknown_signal_is_not_relayed(client: TestClient) -> None:
    cid, patient_token, doctor_token = start_call(client)

    with (
        client.websocket_connect(f"/ws/consultations/{cid}?token={patient_token}") as p,
        client.websocket_connect(f"/ws/consultations/{cid}?token={doctor_token}") as d,
    ):
        p.receive_json()  # joined
        d.receive_json()  # joined
        p.receive_json()  # peer-joined

        p.send_json({"type": "rm -rf"})
        assert p.receive_json() == {"type": "error", "detail": "unsupported signal"}

        p.send_json({"type": "hangup"})
        assert d.receive_json()["type"] == "hangup"


def test_reconnect_replaces_old_socket(client: TestClient) -> None:
    """A page refresh must not lock the participant out of their own call."""
    cid, patient_token, _ = start_call(client)
    url = f"/ws/consultations/{cid}?token={patient_token}"

    with client.websocket_connect(url) as first:
        first.receive_json()
        with client.websocket_connect(url) as second:
            assert second.receive_json()["type"] == "joined"
            with pytest.raises(WebSocketDisconnect) as exc:
                first.receive_json()
            assert exc.value.code == CLOSE_REPLACED


def test_ended_call_cannot_be_rejoined(client: TestClient) -> None:
    cid, patient_token, _ = start_call(client)
    client.post(f"/api/v1/consultations/{cid}/end", json={"token": patient_token})

    with (
        pytest.raises(WebSocketDisconnect) as exc,
        client.websocket_connect(f"/ws/consultations/{cid}?token={patient_token}"),
    ):
        pass
    assert exc.value.code == CLOSE_UNAUTHORIZED
