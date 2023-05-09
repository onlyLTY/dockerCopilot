from . import views
from django.urls import path

app_name = "manager"
urlpatterns = [
    path("", views.manager, name="manager"),
    path("start_container/", views.start_container, name="start_container"),
    path("stop_container/", views.stop_container, name="stop_container"),
    path("rename_container/", views.rename_container, name="rename_container"),
]
