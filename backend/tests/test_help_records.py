"""A professional's record of the people they have helped."""

from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from app import db
from tests.conftest import make_professional, sign_in

HELPED = "/api/v1/consultations/helped"


def request_doctor(client: TestClient, name: str = "Sita") -> dict:
    response = client.post("/api/v1/consultations", json={"patient_name": name})
    assert response.status_code == 201
    return response.json()


def accept(client: TestClient, ticket: dict) -> dict:
    cid = ticket["consultation"]["id"]
    response = client.post(f"/api/v1/consultations/{cid}/accept")
    assert response.status_code == 200
    return response.json()


def end(client: TestClient, ticket: dict) -> None:
    cid = ticket["consultation"]["id"]
    response = client.post(
        f"/api/v1/consultations/{cid}/end", json={"token": ticket["token"]}
    )
    assert response.status_code == 204


def test_a_finished_call_is_counted_with_the_patients_name(client: TestClient) -> None:
    make_professional(client)
    assert client.get(HELPED).json() == {"count": 0, "people": []}

    patient = request_doctor(client, "Sita")
    doctor = accept(client, patient)
    end(client, doctor)

    record = client.get(HELPED).json()
    assert record["count"] == 1
    [person] = record["people"]
    assert person["patient_name"] == "Sita"
    assert person["duration_seconds"] >= 0
    assert person["started_at"] > 0


def test_both_sides_ending_counts_once(client: TestClient) -> None:
    make_professional(client)
    patient = request_doctor(client)
    doctor = accept(client, patient)

    end(client, doctor)
    end(client, patient)  # the other side reports the end too

    assert client.get(HELPED).json()["count"] == 1


def test_a_patient_who_gives_up_waiting_is_not_counted(client: TestClient) -> None:
    make_professional(client)
    end(client, request_doctor(client))

    assert client.get(HELPED).json()["count"] == 0


def test_newest_first_and_counted_per_call(client: TestClient) -> None:
    make_professional(client)
    for name in ("Sita", "Ram", "Gita"):
        end(client, accept(client, request_doctor(client, name)))

    record = client.get(HELPED).json()
    assert record["count"] == 3
    assert [p["patient_name"] for p in record["people"]] == ["Gita", "Ram", "Sita"]


def test_each_professional_sees_only_their_own(client: TestClient) -> None:
    make_professional(client)
    end(client, accept(client, request_doctor(client, "Sita")))

    client.cookies.clear()
    make_professional(client, email="nurse@example.com", name="Maya", role="nurse")
    assert client.get(HELPED).json() == {"count": 0, "people": []}


def test_only_verified_professionals_have_a_record(client: TestClient) -> None:
    assert client.get(HELPED).status_code == 401
    sign_in(client, "patient@example.com")
    client.post("/api/v1/auth/role", json={"role": "patient"})
    assert client.get(HELPED).status_code == 403
    client.cookies.clear()
    make_professional(client, email="new@example.com", status="pending")
    assert client.get(HELPED).status_code == 403


def test_hanging_up_still_works_when_the_database_is_down(
    client: TestClient, monkeypatch: pytest.MonkeyPatch
) -> None:
    make_professional(client)
    doctor = accept(client, request_doctor(client))

    monkeypatch.setattr(db, "_ready", False)
    end(client, doctor)  # still 204
    monkeypatch.setattr(db, "_ready", True)

    assert client.get(HELPED).json()["count"] == 0
