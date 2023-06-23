from django.db import models

# Create your models here.
from django.db import models


class ImageInfo(models.Model):
    # 存储 image 的 id
    image_id = models.CharField(max_length=255, unique=True)

    # 存储本地 image 的创建时间
    local_creation_time = models.DateTimeField(blank=True, null=True)

    # 存储远程 image 的最新更新时间
    remote_last_updated_time = models.DateTimeField(blank=True, null=True)

    # 存储 image 创建出来的容器数量
    container_count = models.IntegerField(default=0)
