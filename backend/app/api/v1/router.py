"""Aggregates every v1 route onto a single router.

Feature routers are added here as the roadmap delivers them:

===========================  =====  ===================================
Router                       Week   Endpoints
===========================  =====  ===================================
``auth``                     3-4    ``/auth/register``, ``/auth/login``,
                                    ``/auth/refresh``
``doctors``                  7-8    ``/doctors``, ``/doctors/availability``,
                                    ``/doctor/match``
``triage``                   9-12   ``/triage``, ``/chat``, ``/history``
``consultations``            13     ``/consultation``, ``/call``,
                                    ``/emergency``
``articles``                 11,18  ``/articles``
``admin``                    16     ``/reports``, ``/doctor/approve``
===========================  =====  ===================================
"""

from fastapi import APIRouter

from app.api.v1 import health

api_router = APIRouter()
api_router.include_router(health.router)
