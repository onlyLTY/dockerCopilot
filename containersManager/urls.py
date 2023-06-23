from . import views
from django.urls import path

app_name = "containers_manager"
urlpatterns = [
    path("", views.containers_manager, name="containersManager"),
    path("start_container/", views.start_container, name="start_container"),
    path("stop_container/", views.stop_container, name="stop_container"),
    path("rename_container/", views.rename_container, name="rename_container"),
    path("create_container/", views.create_container, name="create_container"),
    path("delete_container/", views.delete_container, name="delete_container"),
]
