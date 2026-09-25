from __future__ import annotations

from fastapi.testclient import TestClient


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
