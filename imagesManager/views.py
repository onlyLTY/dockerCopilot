import json
import requests
from .models import ImageInfo
from django.http import JsonResponse
from django.shortcuts import render


# Create your views here.

def images_manager(request):
    images_list = get_images_list(request)
    calculate_container_count_for_image(request)
    images_list = check_image_is_used(images_list)
    return render(
        request,
        "imagesManager/imagesManager.html",
        {
            "images_list": images_list,
        },
    )


def get_images_list(request):
    p = {"all": "true"}
    jwt = request.session['jwt']
    header = {
        "Authorization": jwt
    }
    r = requests.get(
        "http://127.0.0.1:9123/api/endpoints/" + request.session['endpointsId'] + "/docker/images/json",
        headers=header, params=p)
    images_list = r.json()
    spilt_image_name_and_tag(images_list)
    calculate_image_size(images_list)
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


def calculate_image_size(images_list):
    for image in images_list:
        image['Size'] = image['Size'] / 1024 / 1024
        image['Size'] = round(image['Size'], 2)
    return images_list


def get_new_image(request):
    if request.method == "POST":
        # 从POST请求体中获取JSON数据并解析
        data = json.loads(request.body.decode("utf-8"))
        # 从JSON数据中获取名为“num”的值
        image_name_and_tag = data.get("image_name_and_tag")
    else:
        return JsonResponse({"error": "Invalid request method"})
    jwt = request.session['jwt']
    header = {
        "Authorization": jwt
    }
    print("start get new image")
    r = requests.post("http://127.0.0.1:9123/api/endpoints/" + request.session['endpointsId'] +
                      "/docker/images/create?fromImage=" + image_name_and_tag, headers=header)
    print("get new image status code: " + str(r.status_code))
    if r.status_code == 200:
        print("success")
        return JsonResponse({"status": "get_new_image_success"})
    else:
        return JsonResponse({"status": "get_new_image_failed"})


def check_image_is_used(image_list):
    for image in image_list:
        try:
            container_count = ImageInfo.objects.get(image_id=image['Id'].split(":")[1]).container_count
            if container_count > 0:
                image['is_used'] = True
            else:
                image['is_used'] = False
        except ImageInfo.DoesNotExist:
            image['is_used'] = False
    return image_list


def calculate_container_count_for_image(request):
    ImageInfo.objects.all().update(container_count=0)
    container_list = get_container_list(request.session['jwt'], request.session['endpointsId'])
    for container in container_list:
        image_id = container['ImageID'].split(":")[1]
        try:
            image = ImageInfo.objects.get(image_id=image_id)
            image.container_count += 1
            image.save()
        except ImageInfo.DoesNotExist:
            image = ImageInfo(image_id=image_id, container_count=1)
            image.save()


def get_container_list(jwt, endpoints_id):
    p = {"all": "true"}
    header = {
        "Authorization": jwt
    }
    r = requests.get(
        "http://127.0.0.1:9123/api/endpoints/" + endpoints_id + "/docker/containers/json",
        headers=header, params=p)
    container_list = r.json()
    # print(container_list)
    return container_list


def delete_image(request):
    if request.method == "POST":
        # 从POST请求体中获取JSON数据并解析
        data = json.loads(request.body.decode("utf-8"))
        # 从JSON数据中获取名为“num”的值
        num = data.get("num")
    else:
        return JsonResponse({"error": "Invalid request method"})
    num = int(num)
    jwt = request.session['jwt']
    header = {
        "Authorization": jwt
    }
    print("delNum:" + str(num))
    images_list = get_images_list(request)
    r = requests.delete("http://127.0.0.1:9123/api/endpoints/" + request.session['endpointsId'] +
                        "/docker/images/" + images_list[num]['Id'].replace("sha256:", ""),
                        headers=header)
    print("delete:" + r.text)
    print("deleteCode:" + r.status_code.__str__())
    if r.status_code == 200:
        return JsonResponse({"status": "delete_success"})
    elif r.status_code == 409:
        return JsonResponse({"status": "can_not_delete"})
    else:
        return JsonResponse({"status": "delete_failed"})
