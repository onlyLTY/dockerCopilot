from . import views
from django.urls import path

app_name = "manager"
urlpatterns = [
    path("", views.manager, name="manager"),
    path("start_container/", views.start_container, name="start_container"),
    path("stop_container/", views.stop_container, name="stop_container"),
    path("rename_container/", views.rename_container, name="rename_container"),
    path("get_new_image/", views.get_new_image, name="get_new_image"),
    path("create_container/", views.create_container, name="create_container"),
    path("delete_container/", views.delete_container, name="delete_container"),
]
