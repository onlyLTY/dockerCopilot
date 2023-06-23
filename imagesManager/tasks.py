import hashlib
from datetime import datetime
import requests
import os
from django.utils import timezone
from .models import ImageInfo


def check_update():
    jwt = get_jwt()
    endpoints_id = get_endpoints_id(jwt)
    container_list = get_container_list(jwt, endpoints_id)
    container_list = get_image_tag(container_list, jwt, endpoints_id)
    for container in container_list:
        print(container['imageNameAndTag'])
        if container['imageNameAndTag'].split(":")[1] == "None":
            continue
        r = requests.get("https://docker.lieying.fun/v2/repositories/" + container['imageNameAndTag'].split(":")[0] +
                         "/tags/" + container['imageNameAndTag'].split(":")[1])
        remote_image_info = r.json()
        remote_image_create_time = datetime.fromisoformat(remote_image_info['last_updated'].replace("Z", "+00:00"))
        timestamp = container['Created']
        local_image_create_time = timezone.make_aware(datetime.fromtimestamp(timestamp))
        image_id_to_check = container['ImageID'].split(":")[1]
        try:
            image_info = ImageInfo.objects.get(image_id=image_id_to_check)
            image_info.local_creation_time = local_image_create_time
            image_info.remote_last_updated_time = remote_image_create_time
            image_info.save()
        except ImageInfo.DoesNotExist:
            ImageInfo.objects.create(image_id=image_id_to_check,
                                     local_creation_time=local_image_create_time,
                                     remote_last_updated_time=remote_image_create_time)


def get_jwt():
    if os.environ.get('account') is None:
        print("Please set account in environment variable")
        return None
    body = {
        "username": "admin",
        "password": hashlib.md5(os.environ.get('account').encode('utf-8')).hexdigest()
    }
    r = requests.post("http://127.0.0.1:9123/api/auth", json=body)
    jwt = r.json()["jwt"]
    print("jwt: " + jwt)
    return jwt


def get_endpoints_id(jwt):
    header = {
        "Authorization": jwt
    }
    r = requests.get("http://127.0.0.1:9123/api/endpoints",
                     headers=header)
    info = r.json()
    print(info)
    endpoints_id = str(info[0]['Id'])
    return endpoints_id


def get_container_list(jwt, endpoints_id):
    p = {"all": "true"}
    header = {
        "Authorization": jwt
    }
    r = requests.get(
        "http://127.0.0.1:9123/api/endpoints/" + endpoints_id + "/docker/containers/json",
        headers=header, params=p)
    container_list = r.json()
    container_list = get_image_tag(container_list, jwt, endpoints_id)
    print(container_list)
    return container_list


def get_image_tag(container_list, jwt, endpoints_id):
    for container in container_list:
        image_id = container['ImageID'].split(":")[1]
        images_list = get_images_list(jwt, endpoints_id)
        images_dict = create_image_id_map(images_list)
        container['imageNameAndTag'] = \
            images_dict.get(image_id)['image_name'] + ":" + images_dict.get(image_id)['image_tag']

    return container_list


def create_image_id_map(images_list):
    # Create a mapping from image_id to the image data
    return {image['Id'].split(':')[1]: image for image in images_list}


def get_images_list(jwt, endpoints_id):
    p = {"all": "true"}
    header = {
        "Authorization": jwt
    }
    r = requests.get(
        "http://127.0.0.1:9123/api/endpoints/" + endpoints_id + "/docker/images/json",
        headers=header, params=p)
    images_list = r.json()
    spilt_image_name_and_tag(images_list)
    return images_list


def spilt_image_name_and_tag(images_list):
    for image in images_list:
        if image.get('RepoTags'):
            image['image_name'] = image['RepoTags'][0].split(":")[0]
            image['image_tag'] = image['RepoTags'][0].split(":")[1]
        else:
            image['image_name'] = image['RepoDigests'][0].split("@")[0]
            image['image_tag'] = "None"
    return images_list
