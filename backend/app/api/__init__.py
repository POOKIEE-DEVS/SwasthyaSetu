"""HTTP routers."""

from fastapi import APIRouter

from app.api import auth, chat, consultations, health

api_router = APIRouter()
api_router.include_router(health.router)
api_router.include_router(chat.router)
api_router.include_router(consultations.router)
api_router.include_router(auth.router)
