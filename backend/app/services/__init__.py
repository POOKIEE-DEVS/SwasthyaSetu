"""Domain logic -- the layer between HTTP routing and the database.

Routers in ``app.api`` handle encoding and dispatch only; the rules live
here, so the same operation can be driven by an HTTP request, a WebSocket
frame, or a Celery task without being reimplemented.

======================  ====  ==============================================
Module                  Week  Responsibility
======================  ====  ==============================================
``auth``                3-4   Registration, login, JWT + refresh token
                              issuance, role-based access control
``matching``            7-8   Rank available doctors by location,
                              specialization, availability, language, and
                              experience; assign one
``consultation``        13    Consultation lifecycle and WebRTC room setup
``notification``        14    Compose and queue email / SMS / push messages
``admin``               16    Doctor approval, content management, reports
======================  ====  ==============================================
"""
