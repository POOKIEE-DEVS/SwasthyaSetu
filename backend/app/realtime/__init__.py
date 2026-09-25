"""Real-time communication: WebSockets and WebRTC signalling.

Two distinct jobs share this package:

- **WebSockets** carry application events -- the doctor's live consultation
  queue, notification broadcasts, and triage progress -- over a connection
  FastAPI serves natively, with no separate message broker in the request
  path.
- **WebRTC** carries the consultation's own audio and video peer-to-peer.
  The server never sees media; it only brokers the SDP offer/answer and ICE
  candidate exchange, and hands out TURN credentials for the cases where a
  direct peer connection cannot be established.
"""

from app.realtime.connection_manager import ConnectionManager, manager

__all__ = ["ConnectionManager", "manager"]
