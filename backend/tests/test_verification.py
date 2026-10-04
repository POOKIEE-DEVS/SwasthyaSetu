from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from tests.conftest import ADMIN_EMAIL, sign_in

PNG = b"\x89PNG\r\n\x1a\n" + b"\x00" * 64
JPEG = b"\xff\xd8\xff\xe0" + b"\x00" * 64
PDF = b"%PDF-1.7\n" + b"\x00" * 64
SCRIPT = b"<script>alert(1)</script>"


def doctor_form(**overrides: object) -> tuple[dict, dict]:
    data = {
        "role": "doctor",
        "full_name": "Dr. Anita Karki",
        "phone": "+977 9841234567",
        "citizenship_number": "27-01-71-12345",
        "citizenship_district": "Kathmandu",
        "council_number": "12345",
        "consent": "true",
        **overrides,
    }
    files = {
        "citizenship_front": ("front.png", PNG, "image/png"),
        "citizenship_back": ("back.jpg", JPEG, "image/jpeg"),
        "council_certificate": ("nmc.pdf", PDF, "application/pdf"),
    }
    return data, files


def student_form() -> tuple[dict, dict]:
    data = {
        "role": "student",
        "full_name": "Bikash Thapa",
        "phone": "9800000000",
        "citizenship_number": "45-02-75-00001",
        "citizenship_district": "Kaski",
        "institution": "Manipal College of Medical Sciences",
        "recommender_name": "Dr. Anita Karki",
        "recommender_nmc": "12345",
        "consent": "true",
    }
    files = {
        "citizenship_front": ("front.png", PNG, "image/png"),
        "citizenship_back": ("back.png", PNG, "image/png"),
        "recommendation_letter": ("letter.pdf", PDF, "application/pdf"),
    }
    return data, files


def apply(client: TestClient, data: dict, files: dict):
    return client.post("/api/v1/applications", data=data, files=files)


def as_admin(client: TestClient) -> None:
    client.cookies.clear()
    sign_in(client, ADMIN_EMAIL, "Admin")


def test_must_be_signed_in(client: TestClient) -> None:
    assert apply(client, *doctor_form()).status_code == 401


def test_doctor_applies_and_sees_pending(client: TestClient) -> None:
    sign_in(client, "anita@example.com", "Anita")

    response = apply(client, *doctor_form())

    assert response.status_code == 201, response.text
    body = response.json()
    assert body["status"] == "pending"
    assert body["council_number"] == "12345"
    assert sorted(d["kind"] for d in body["documents"]) == [
        "citizenship_back",
        "citizenship_front",
        "council_certificate",
    ]
    # Types come from the bytes.
    by_kind = {d["kind"]: d["content_type"] for d in body["documents"]}
    assert by_kind["council_certificate"] == "application/pdf"

    me = client.get("/api/v1/auth/me").json()["user"]
    assert me["role"] == "doctor"
    assert me["verification"]["status"] == "pending"
    assert client.get("/api/v1/applications/me").json()["id"] == body["id"]


@pytest.mark.parametrize(
    ("role", "council"),
    [
        ("pharmacist", "Nepal Pharmacy Council"),
        ("nurse", "Nepal Nursing Council"),
        ("paramedic", "Nepal Health Professional Council"),
    ],
)
def test_registered_professions_need_their_council_number_and_certificate(
    client: TestClient, role: str, council: str
) -> None:
    sign_in(client, f"{role}@example.com")
    data, files = doctor_form(role=role, council_number="")
    missing_number = apply(client, data, files)
    assert missing_number.status_code == 422
    assert council in missing_number.json()["detail"]

    data, files = doctor_form(role=role)
    del files["council_certificate"]
    missing_certificate = apply(client, data, files)
    assert missing_certificate.status_code == 422
    assert "certificate" in missing_certificate.json()["detail"]

    data, files = doctor_form(role=role)
    submitted = apply(client, data, files)
    assert submitted.status_code == 201, submitted.text
    assert submitted.json()["role"] == role
    assert submitted.json()["council_number"]


def test_student_needs_recommendation(client: TestClient) -> None:
    sign_in(client, "student@example.com")
    data, files = student_form()
    del files["recommendation_letter"]
    assert "recommendation" in apply(client, data, files).json()["detail"]

    data, files = student_form()
    response = apply(client, data, files)
    assert response.status_code == 201
    assert response.json()["recommender_nmc"] == "12345"


@pytest.mark.parametrize(
    ("field", "value", "message"),
    [
        ("phone", "abc", "phone"),
        ("citizenship_number", "", "citizenship number"),
        ("consent", "false", "agree"),
    ],
)
def test_field_validation(
    client: TestClient, field: str, value: str, message: str
) -> None:
    sign_in(client, "x@example.com")
    response = apply(client, *doctor_form(**{field: value}))
    assert response.status_code == 422
    assert message in response.json()["detail"]


def test_disguised_file_is_refused(client: TestClient) -> None:
    sign_in(client, "x@example.com")
    data, files = doctor_form()
    files["citizenship_front"] = ("front.jpg", SCRIPT, "image/jpeg")
    response = apply(client, data, files)
    assert response.status_code == 422
    assert "Citizenship (front)" in response.json()["detail"]


def test_oversized_file_is_refused(
    client: TestClient, monkeypatch: pytest.MonkeyPatch
) -> None:
    from app.core.config import settings

    monkeypatch.setattr(settings, "max_upload_bytes", 1024 * 1024)
    sign_in(client, "x@example.com")
    data, files = doctor_form()
    files["citizenship_back"] = ("big.png", PNG + b"\x00" * (1024 * 1024), "image/png")
    assert "too large" in apply(client, data, files).json()["detail"]


def test_selfie_must_be_a_photo(client: TestClient) -> None:
    sign_in(client, "x@example.com")
    data, files = doctor_form()
    files["selfie"] = ("selfie.pdf", PDF, "application/pdf")
    assert "photo" in apply(client, data, files).json()["detail"]


def test_admin_only(client: TestClient) -> None:
    sign_in(client, "anita@example.com")
    assert client.get("/api/v1/admin/applications").status_code == 403
    client.cookies.clear()
    assert client.get("/api/v1/admin/applications").status_code == 401


def test_admin_reviews_documents_and_approves(client: TestClient) -> None:
    sign_in(client, "anita@example.com", "Anita")
    application_id = apply(client, *doctor_form()).json()["id"]

    as_admin(client)
    queue = client.get("/api/v1/admin/applications").json()
    assert [a["id"] for a in queue] == [application_id]
    card = queue[0]
    assert card["user_email"] == "anita@example.com"
    assert [h["action"] for h in card["history"]] == ["submitted"]

    front = next(d for d in card["documents"] if d["kind"] == "citizenship_front")
    image = client.get(f"/api/v1/admin/documents/{front['id']}")
    assert image.content == PNG
    assert image.headers["content-type"] == "image/png"
    assert image.headers["x-content-type-options"] == "nosniff"
    assert "no-store" in image.headers["cache-control"]

    approved = client.post(f"/api/v1/admin/applications/{application_id}/approve")
    assert approved.json()["status"] == "approved"
    assert [h["action"] for h in approved.json()["history"]] == [
        "submitted",
        "approved",
    ]
    assert client.get("/api/v1/admin/applications").json() == []
    approved_list = client.get("/api/v1/admin/applications?status=approved").json()
    assert [a["id"] for a in approved_list] == [application_id]


def test_applicant_cannot_read_documents(client: TestClient) -> None:
    sign_in(client, "anita@example.com")
    doc_id = apply(client, *doctor_form()).json()["documents"][0]["id"]
    assert client.get(f"/api/v1/admin/documents/{doc_id}").status_code == 403


def test_rejected_applicant_fixes_and_resubmits(client: TestClient) -> None:
    sign_in(client, "anita@example.com", "Anita")
    application_id = apply(client, *doctor_form()).json()["id"]

    as_admin(client)
    rejected = client.post(
        f"/api/v1/admin/applications/{application_id}/reject",
        json={"reason": "Citizenship photo is blurry."},
    )
    assert rejected.json()["status"] == "rejected"

    client.cookies.clear()
    sign_in(client, "anita@example.com")
    me = client.get("/api/v1/auth/me").json()["user"]
    assert me["verification"]["rejection_reason"] == "Citizenship photo is blurry."

    again = apply(client, *doctor_form(citizenship_district="Lalitpur"))
    assert again.status_code == 201
    assert again.json()["id"] == application_id  # same application, updated
    assert again.json()["status"] == "pending"
    assert again.json()["rejection_reason"] is None
    assert len(again.json()["documents"]) == 3  # old documents replaced

    as_admin(client)
    card = client.get("/api/v1/admin/applications").json()[0]
    assert [h["action"] for h in card["history"]] == [
        "submitted",
        "rejected",
        "resubmitted",
    ]


def test_reject_needs_a_reason(client: TestClient) -> None:
    sign_in(client, "anita@example.com")
    application_id = apply(client, *doctor_form()).json()["id"]
    as_admin(client)
    response = client.post(
        f"/api/v1/admin/applications/{application_id}/reject", json={"reason": ""}
    )
    assert response.status_code == 422


def test_approved_cannot_resubmit_or_switch_role(client: TestClient) -> None:
    sign_in(client, "anita@example.com")
    application_id = apply(client, *doctor_form()).json()["id"]
    as_admin(client)
    client.post(f"/api/v1/admin/applications/{application_id}/approve")

    client.cookies.clear()
    sign_in(client, "anita@example.com")
    assert apply(client, *doctor_form()).status_code == 409
    switch = client.post("/api/v1/auth/role", json={"role": "pharmacist"})
    assert switch.status_code == 409


def test_admin_can_revoke_an_approval(client: TestClient) -> None:
    sign_in(client, "anita@example.com")
    application_id = apply(client, *doctor_form()).json()["id"]
    as_admin(client)
    client.post(f"/api/v1/admin/applications/{application_id}/approve")
    revoked = client.post(
        f"/api/v1/admin/applications/{application_id}/reject",
        json={"reason": "NMC registration not found."},
    )
    assert revoked.json()["status"] == "rejected"


def test_both_citizenship_sides_required(client: TestClient) -> None:
    sign_in(client, "x@example.com")
    data, files = doctor_form()
    del files["citizenship_back"]
    response = apply(client, data, files)
    assert response.status_code == 422
    assert "both sides" in response.json()["detail"]
