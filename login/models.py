from django.db import models


# Create your models here.

class User(models.Model):
    cookie = models.CharField(max_length=1000)
    phone = models.CharField(max_length=100)
    nasIp = models.CharField(max_length=100)
