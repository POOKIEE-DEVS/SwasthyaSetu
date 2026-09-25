"""In-process registry of live WebSocket connections, grouped into rooms.

A "room" is a fan-out target: ``consultation:{id}`` for one call's
signalling, or ``doctors:queue`` for the live queue every online doctor
watches.

This registry lives in one process, which is why the app must run as a
single worker. Scaling beyond that would mean fanning broadcasts out through
a shared broker (e.g. Redis pub/sub); :meth:`broadcast` is where that would
go.
"""

from __future__ import annotations

import asyncio
import logging
from collections import defaultdict
from typing import Any

from fastapi import WebSocket

logger = logging.getLogger(__name__)


class ConnectionManager:
    """Tracks open sockets per room and fans messages out to them."""

    def __init__(self) -> None:
        self._rooms: dict[str, set[WebSocket]] = defaultdict(set)
        # Guards the room map: connect/disconnect run concurrently with
        # broadcasts, and a set mutated mid-iteration raises.
        self._lock = asyncio.Lock()

    async def connect(self, room: str, websocket: WebSocket) -> None:
        """Accept the handshake and register the socket."""
        await websocket.accept()
        async with self._lock:
            self._rooms[room].add(websocket)
        logger.info(
            "websocket connected", extra={"room": room, "size": self.room_size(room)}
        )

    async def disconnect(self, room: str, websocket: WebSocket) -> None:
        """Deregister a socket, dropping the room once it is empty."""
        async with self._lock:
            self._rooms[room].discard(websocket)
            if not self._rooms[room]:
                del self._rooms[room]
        logger.info(
            "websocket disconnected", extra={"room": room, "size": self.room_size(room)}
        )

    def room_size(self, room: str) -> int:
        return len(self._rooms.get(room, ()))

    async def send(self, websocket: WebSocket, message: dict[str, Any]) -> None:
        await websocket.send_json(message)

    async def broadcast(
        self,
        room: str,
        message: dict[str, Any],
        *,
        exclude: WebSocket | None = None,
    ) -> None:
        """Send ``message`` to every socket in ``room``.

        ``exclude`` skips the originating socket, which is what signalling
        wants: a peer should not receive its own offer back.

        Sends run concurrently and failures are collected rather than raised
        -- one dead socket must not stop an emergency broadcast reaching the
        others. Sockets that fail are dropped, since a send error means the
        peer is already gone.
        """
        async with self._lock:
            targets = [ws for ws in self._rooms.get(room, ()) if ws is not exclude]

        if not targets:
            return

        results = await asyncio.gather(
            *(ws.send_json(message) for ws in targets), return_exceptions=True
        )

        dead = [
            ws
            for ws, outcome in zip(targets, results, strict=True)
            if isinstance(outcome, BaseException)
        ]
        if dead:
            logger.warning(
                "dropping unreachable sockets",
                extra={"room": room, "count": len(dead)},
            )
            async with self._lock:
                self._rooms[room].difference_update(dead)
                if not self._rooms[room]:
                    self._rooms.pop(room, None)


# Shared across the process; routers import this rather than constructing
# their own, which would fragment the room map.
manager = ConnectionManager()
