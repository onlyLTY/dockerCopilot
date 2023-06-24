import hashlib
import json
import os
import time
from datetime import datetime
import requests
from django.http import HttpResponse, JsonResponse
from imagesManager.models import ImageInfo
from django.template import loader
from django.shortcuts import render, redirect
from imagesManager.views import get_images_list


# Create your views here.
def containers_manager(request):
    container_list = get_container_list(request)
    get_images_list(request)
    check_update(container_list)
    return render(
        request,
        "containersManager/containersManager.html",
        {
            "container_list": container_list,
        },
    )


def get_container_list(request):
    p = {"all": "true"}
    jwt = request.session.get('jwt')
    header = {
        "Authorization": jwt
    }
    r = requests.get(
        "http://127.0.0.1:9123/api/endpoints/" + request.session['endpointsId'] + "/docker/containers/json",
        headers=header, params=p)
    container_list = r.json()
    container_list = get_image_tag(request, container_list)
    # print(container_list)
    return container_list


def check_update(container_list):
    image_update_info = ImageInfo.objects.all()
    for container in container_list:
        container_image_id = container['ImageID'].split(":")[1]
        try:
            if image_update_info.filter(image_id=container_image_id).exists():
                if image_update_info.get(image_id=container_image_id).remote_last_updated_time > \
                        image_update_info.get(image_id=container_image_id).local_creation_time:
                    container['update'] = True
                else:
                    container['update'] = False
            else:
                container['update'] = False
        except TypeError:
            container['update'] = False


def get_image_tag(request, container_list):
    for container in container_list:
        image_id = container['ImageID'].split(":")[1]
        images_list = get_images_list(request)
        images_dict = create_image_id_map(images_list)
        container['imageNameAndTag'] = \
            images_dict.get(image_id)['image_name'] + ":" + images_dict.get(image_id)['image_tag']
    return container_list


def create_image_id_map(images_list):
    # Create a mapping from image_id to the image data
    return {image['Id'].split(':')[1]: image for image in images_list}


def start_container(request):
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
    container_list = get_container_list(request)
    r = requests.post("http://127.0.0.1:9123/api/endpoints/" + request.session['endpointsId'] +
                      "/docker/containers/" + container_list[num]['Id'].replace("sha256:", "") + "/start",
                      headers=header)
    #print("start_container" + str(r.status_code))
    #print("start_container" + r.text)
    if r.status_code == 204:
        return JsonResponse({"status": "start_success"})
    else:
        return JsonResponse({"status": "start_failed"})


def stop_container(request):
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
    container_list = get_container_list(request)
    r = requests.post("http://127.0.0.1:9123/api/endpoints/" + request.session['endpointsId'] +
                      "/docker/containers/" + container_list[num]['Id'].replace("sha256:", "") + "/stop",
                      headers=header)
    #print(r.status_code)
    if r.status_code == 204:
        return JsonResponse({"status": "stop_success"})
    else:
        return JsonResponse({"status": "stop_failed"})


def get_containers_info(header, endpointsId, containers_list, num):
    r = requests.get("http://127.0.0.1:9123/api/endpoints/" + endpointsId +
                     "/docker/containers/" + containers_list[num]['Id'].replace("sha256:", "") + "/json"
                     , headers=header)
    info = r.json()
    return info


def rename_container(request):
    if request.method == "POST":
        # 从POST请求体中获取JSON数据并解析
        data = json.loads(request.body.decode("utf-8"))
        # 从JSON数据中获取名为“num”的值
        num = data.get("num")
        new_name = data.get("new_name")
    else:
        return JsonResponse({"error": "Invalid request method"})
    num = int(num)
    jwt = request.session['jwt']
    header = {
        "Authorization": jwt
    }
    container_list = get_container_list(request)
    r = requests.post("http://127.0.0.1:9123/api/endpoints/" + request.session['endpointsId'] +
                      "/docker/containers/" + container_list[num]['Id'].replace("sha256:", "") + "/rename"
                                                                                                 "?name=" +
                      new_name, headers=header)
    #print(r.text)
    if r.status_code == 204:
        return JsonResponse({"status": "rename_success"})
    else:
        return JsonResponse({"status": "rename_failed"})


def create_container(request):
    if request.method == "POST":
        # 从POST请求体中获取JSON数据并解析
        data = json.loads(request.body.decode("utf-8"))
        # 从JSON数据中获取名为“num”的值
        num = data.get("num")
        container_name = data.get("name")
        image_name_and_tag = data.get("image_name_and_tag")
    else:
        return JsonResponse({"error": "Invalid request method"})
    num = int(num)
    jwt = request.session['jwt']
    header = {
        "Authorization": jwt
    }
    body = {}
    container_info = get_containers_info(header,
                                         request.session['endpointsId'], get_container_list(request), num)
    #print("---------------------------------")
    #print(container_info)
    for i in container_info['Config']:
        body[i] = container_info['Config'][i]
    body['HostConfig'] = container_info['HostConfig']
    body['name'] = container_name
    body['NetworkingConfig'] = {}
    body['NetworkingConfig']['EndpointsConfig'] = {}
    if 'bridge' in container_info['NetworkSettings']['Networks']:
        body['NetworkingConfig']['EndpointsConfig']['bridge'] = \
            container_info['NetworkSettings']['Networks']['bridge']
    if 'host' in container_info['NetworkSettings']['Networks']:
        body['NetworkingConfig']['EndpointsConfig']['host'] = \
            container_info['NetworkSettings']['Networks']['host']
    #print("---------------------------------")
    body['Image'] = image_name_and_tag
    #print(body['Image'])
    #print("---------------------------------")
    r = requests.post("http://127.0.0.1:9123/api/endpoints/" + request.session['endpointsId'] +
                      "/docker/containers/create?name=" + container_name, headers=header, json=body)
    #print("create:" + str(r.status_code))
    #print("create:" + r.text)
    if r.status_code == 200:

        return JsonResponse({"status": "create_success"})
    else:
        return JsonResponse({"status": "create_failed"})


def delete_container(request):
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
    #print("delNum:" + str(num))
    container_list = get_container_list(request)
    r = requests.delete("http://127.0.0.1:9123/api/endpoints/" + request.session['endpointsId'] +
                        "/docker/containers/" + container_list[num]['Id'].replace("sha256:", "") + "?v=1",
                        headers=header)
    #print("delete:" + r.text)
    if r.status_code == 204:
        return JsonResponse({"status": "delete_success"})
    else:
        return JsonResponse({"status": "delete_failed"})
