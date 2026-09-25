"""HTTP and WebSocket transport: routing, encoding, and dependencies.

This layer stays free of business logic -- that belongs in ``app.services``
and ``app.ai``, which the routers only dispatch to.
"""
