from django.urls import path

from . import views
from django.urls import path
from .views import check_device, verification_page

app_name = "login"
urlpatterns = [
    path('', check_device, name='check_device'),
    path('verify', verification_page, name='verification_page'),
]
