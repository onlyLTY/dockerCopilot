from . import views
from django.urls import path

app_name = "images_manager"
urlpatterns = [
    path("", views.images_manager, name="imagesManager"),
    path("get_new_image/", views.get_new_image, name="get_new_image"),
    path("delete_image/", views.delete_image, name="delete_image"),
]