from __future__ import annotations

import pytest
from fastapi.testclient import TestClient
from starlette.websockets import WebSocketDisconnect

from app.realtime.websocket import CLOSE_NOT_VERIFIED
from tests.conftest import DOCTOR_NAME, make_professional, sign_in


@pytest.fixture(autouse=True)
def verified_doctor(client: TestClient) -> None:
    """The same client acts as patient (no login needed) and as a verified
    doctor (signed in)."""
    make_professional(client)


def request_doctor(client: TestClient, name: str = "Sita") -> dict:
    response = client.post(
        "/api/v1/consultations",
        json={"patient_name": name, "summary": "Fever for two days"},
    )
    assert response.status_code == 201
    return response.json()


def test_patient_request_returns_ticket(client: TestClient) -> None:
    ticket = request_doctor(client)

    assert ticket["role"] == "patient"
    assert ticket["token"]
    assert ticket["consultation"]["status"] == "waiting"
    assert ticket["ice_servers"][0]["urls"][0].startswith("stun:")


def test_waiting_list_never_exposes_tokens(client: TestClient) -> None:
    request_doctor(client)

    waiting = client.get("/api/v1/consultations").json()

    assert len(waiting) == 1
    assert waiting[0]["patient_name"] == "Sita"
    assert "token" not in waiting[0]


def test_doctor_accepts(client: TestClient) -> None:
    ticket = request_doctor(client)
    cid = ticket["consultation"]["id"]

    accepted = client.post(f"/api/v1/consultations/{cid}/accept")

    assert accepted.status_code == 200
    body = accepted.json()
    assert body["role"] == "doctor"
    assert body["token"] != ticket["token"]
    assert body["consultation"]["summary"] == "Fever for two days"
    # The badge the patient will see.
    assert body["consultation"]["professional"] == {
        "name": DOCTOR_NAME,
        "role": "doctor",
    }
    assert client.get("/api/v1/consultations").json() == []


def test_second_accept_conflicts(client: TestClient) -> None:
    cid = request_doctor(client)["consultation"]["id"]
    client.post(f"/api/v1/consultations/{cid}/accept")

    assert client.post(f"/api/v1/consultations/{cid}/accept").status_code == 409


def test_accept_unknown_is_404(client: TestClient) -> None:
    assert client.post("/api/v1/consultations/nope/accept").status_code == 404


def test_end_requires_a_participant_token(client: TestClient) -> None:
    ticket = request_doctor(client)
    cid = ticket["consultation"]["id"]

    wrong = client.post(f"/api/v1/consultations/{cid}/end", json={"token": "x"})
    right = client.post(
        f"/api/v1/consultations/{cid}/end", json={"token": ticket["token"]}
    )

    assert wrong.status_code == 404
    assert right.status_code == 204
    assert client.get("/api/v1/consultations").json() == []


def test_patient_needs_no_login(client: TestClient) -> None:
    client.cookies.clear()
    assert request_doctor(client)["role"] == "patient"


def test_signed_out_cannot_see_or_accept(client: TestClient) -> None:
    cid = request_doctor(client)["consultation"]["id"]
    client.cookies.clear()
    assert client.get("/api/v1/consultations").status_code == 401
    assert client.post(f"/api/v1/consultations/{cid}/accept").status_code == 401


def test_patient_account_cannot_accept(client: TestClient) -> None:
    cid = request_doctor(client)["consultation"]["id"]
    client.cookies.clear()
    sign_in(client, "patient@example.com")
    client.post("/api/v1/auth/role", json={"role": "patient"})
    assert client.get("/api/v1/consultations").status_code == 403
    assert client.post(f"/api/v1/consultations/{cid}/accept").status_code == 403


@pytest.mark.parametrize("status", ["pending", "rejected"])
def test_unverified_professional_cannot_accept(client: TestClient, status: str) -> None:
    cid = request_doctor(client)["consultation"]["id"]
    client.cookies.clear()
    make_professional(client, email="new@example.com", status=status)
    response = client.post(f"/api/v1/consultations/{cid}/accept")
    assert response.status_code == 403
    assert "verified" in response.json()["detail"]


@pytest.mark.parametrize("role", ["pharmacist", "student"])
def test_verified_pharmacists_and_students_can_accept(
    client: TestClient, role: str
) -> None:
    cid = request_doctor(client)["consultation"]["id"]
    client.cookies.clear()
    make_professional(client, email=f"{role}@example.com", name="Hari", role=role)
    accepted = client.post(f"/api/v1/consultations/{cid}/accept").json()
    assert accepted["consultation"]["professional"] == {"name": "Hari", "role": role}


def test_queue_socket_requires_verification(client: TestClient) -> None:
    client.cookies.clear()
    with (
        pytest.raises(WebSocketDisconnect) as exc,
        client.websocket_connect("/ws/doctors"),
    ):
        pass
    assert exc.value.code == CLOSE_NOT_VERIFIED
