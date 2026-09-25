"""HTTP routers."""

from fastapi import APIRouter

from app.api import chat, consultations, health

api_router = APIRouter()
api_router.include_router(health.router)
api_router.include_router(chat.router)
api_router.include_router(consultations.router)
