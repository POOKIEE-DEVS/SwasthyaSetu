"""Password hashing and JWT issuance/verification.

Implemented in Week 3 (auth core) and Week 4 (refresh tokens + RBAC).
Kept as a named module from Week 1 so that no other layer is ever tempted to
grow its own copy of token or password handling.

Planned surface:

- ``hash_password`` / ``verify_password`` — argon2 via passlib.
- ``create_access_token`` / ``create_refresh_token`` — short-lived access
  token paired with a long-lived rotating refresh token.
- ``decode_token`` — signature and expiry verification, raising on failure.
"""
