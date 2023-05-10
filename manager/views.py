import json

import requests
from django.http import HttpResponse, JsonResponse
from django.template import loader
from login.models import User
from django.shortcuts import render, redirect


# Create your views here.
def manager(request):
    request.session['jwt'] = request.COOKIES.get("jwt")
    request.session['cookie'] = User.objects.get(pk=1).cookie
    request.session['nasIp'] = User.objects.get(pk=1).nasIp
    request.session['phone'] = User.objects.get(pk=1).phone
    try:
        get_endpoints_ID(request)
    except:
        redirect("login:check")
    container_list = get_container_list(request)
    return render(
        request,
        "manager/manager.html",
        {
            "container_list": container_list,
        },
    )


def get_endpoints_ID(request):
    nas_ip = request.session['nasIp']
    cookie = request.session['cookie']
    jwt = request.session['jwt']
    header = {
        'Cookie': cookie,
        "Authorization": jwt
    }
    r = requests.get("http://" + nas_ip + ":5055/docker/api/endpoints",
                     headers=header)
    info = r.json()
    request.session['endpointsId'] = str(info[0]['Id'])


def get_container_list(request):
    p = {"all": "true"}
    nas_ip = request.session['nasIp']
    cookie = request.session['cookie']
    jwt = request.session['jwt']
    header = {
        'Cookie': cookie,
        "Authorization": jwt
    }
    r = requests.get(
        "http://" + nas_ip + ":5055/docker/api/endpoints/" + request.session['endpointsId'] + "/docker/containers/json",
        headers=header, params=p)
    container_list = r.json()
    # print(container_list)
    return container_list


def start_container(request):
    if request.method == "POST":
        # 从POST请求体中获取JSON数据并解析
        data = json.loads(request.body.decode("utf-8"))
        # 从JSON数据中获取名为“num”的值
        num = data.get("num")
    else:
        return JsonResponse({"error": "Invalid request method"})
    num = int(num)
    nas_ip = request.session['nasIp']
    cookie = request.session['cookie']
    jwt = request.session['jwt']
    header = {
        'Cookie': cookie,
        "Authorization": jwt
    }
    container_list = get_container_list(request)
    r = requests.post("http://" + nas_ip + ":5055/docker/api/endpoints/" + request.session['endpointsId'] +
                      "/docker/containers/" + container_list[num]['Id'].replace("sha256:", "") + "/start",
                      headers=header)
    print(r.status_code)
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
    nas_ip = request.session['nasIp']
    cookie = request.session['cookie']
    jwt = request.session['jwt']
    header = {
        'Cookie': cookie,
        "Authorization": jwt
    }
    container_list = get_container_list(request)
    r = requests.post("http://" + nas_ip + ":5055/docker/api/endpoints/" + request.session['endpointsId'] +
                      "/docker/containers/" + container_list[num]['Id'].replace("sha256:", "") + "/stop",
                      headers=header)
    print(r.status_code)
    if r.status_code == 204:
        return JsonResponse({"status": "stop_success"})
    else:
        return JsonResponse({"status": "stop_failed"})


def get_containers_info(nas_ip, header, endpointsId, containers_list, num):
    r = requests.get("http://" + nas_ip + ":5055/docker/api/endpoints/" + endpointsId +
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
    nas_ip = request.session['nasIp']
    cookie = request.session['cookie']
    jwt = request.session['jwt']
    header = {
        'Cookie': cookie,
        "Authorization": jwt
    }
    container_list = get_container_list(request)
    r = requests.post("http://" + nas_ip + ":5055/docker/api/endpoints/" + request.session['endpointsId'] +
                      "/docker/containers/" + container_list[num]['Id'].replace("sha256:", "") + "/rename"
                                                                                                 "?name=" +
                      new_name, headers=header)
    print(r.text)
    if r.status_code == 204:
        return JsonResponse({"status": "rename_success"})
    else:
        return JsonResponse({"status": "rename_failed"})


def get_new_image(request):
    if request.method == "POST":
        # 从POST请求体中获取JSON数据并解析
        data = json.loads(request.body.decode("utf-8"))
        # 从JSON数据中获取名为“num”的值
        image_name_and_tag = data.get("image_name_and_tag")
    else:
        return JsonResponse({"error": "Invalid request method"})
    nas_ip = request.session['nasIp']
    cookie = request.session['cookie']
    jwt = request.session['jwt']
    header = {
        'Cookie': cookie,
        "Authorization": jwt
    }
    print("start get new image")
    r = requests.post("http://" + nas_ip + ":5055/docker/api/endpoints/" + request.session['endpointsId'] +
                      "/docker/images/create?fromImage=" + image_name_and_tag, headers=header)
    print("get new image status code: " + str(r.status_code))
    if r.status_code == 200:
        print("success")
        return JsonResponse({"status": "get_new_image_success"})
    else:
        return JsonResponse({"status": "get_new_image_failed"})


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
    nas_ip = request.session['nasIp']
    cookie = request.session['cookie']
    jwt = request.session['jwt']
    header = {
        'Cookie': cookie,
        "Authorization": jwt
    }
    body = {}
    container_info = get_containers_info(nas_ip, header,
                                         request.session['endpointsId'], get_container_list(request), num)
    print("---------------------------------")
    print(container_info)
    for i in container_info['Config']:
        body[i] = container_info['Config'][i]
    body['HostConfig'] = container_info['HostConfig']
    body['name'] = container_name
    body['NetworkingConfig'] = {}
    body['NetworkingConfig']['EndpointsConfig'] = {}
    body['NetworkingConfig']['EndpointsConfig']['bridge'] = \
        container_info['NetworkSettings']['Networks']['bridge']
    print("---------------------------------")
    body['Image'] = image_name_and_tag
    print(body['Image'])
    print("---------------------------------")
    r = requests.post("http://" + nas_ip + ":5055/docker/api/endpoints/" + request.session['endpointsId'] +
                      "/docker/containers/create?name=" + container_name, headers=header, json=body)
    print("create:" + str(r.status_code))
    print("create:" + r.text)
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
    nas_ip = request.session['nasIp']
    cookie = request.session['cookie']
    jwt = request.session['jwt']
    header = {
        'Cookie': cookie,
        "Authorization": jwt
    }
    print("delNum:" + str(num))
    container_list = get_container_list(request)
    r = requests.delete("http://" + nas_ip + ":5055/docker/api/endpoints/" + request.session['endpointsId'] +
                        "/docker/containers/" + container_list[num]['Id'].replace("sha256:", "") + "?v=1",
                        headers=header)
    print("delete:" + r.text)
    if r.status_code == 204:
        return JsonResponse({"status": "delete_success"})
    else:
        return JsonResponse({"status": "delete_failed"})


class DockerUpdate:
    account = None
    cookie = None
    jwt = None
    endpointsId = None
    containers_list = None

    def __init__(self, account, cookie, jwt, nas_ip):
        self.account = account
        self.cookie = cookie.strip()
        self.jwt = "Bearer " + jwt
        self.nas_ip = nas_ip
        self.header = {
            'Cookie': self.cookie,
            "Authorization": self.jwt
        }
        self.body = {}
        # print("----")
        # print(self.jwt)

    def get_endpoints_ID(self):
        r = requests.get("http://" + self.nas_ip + ":5055/docker/api/endpoints",
                         headers=self.header)
        info = r.json()
        print(r.status_code)
        self.endpointsId = info[0]['Id']

    def get_containers_list(self):
        p = {"all": "true"}
        r = requests.get("http://" + self.nas_ip + ":5055/docker/api/endpoints/" + str(self.endpointsId) +
                         "/docker/containers/json", headers=self.header, params=p)
        # print(self.header)
        # print(r.json())
        self.containers_list = r.json()
        # print(len(info))
        # print(info[1]['Mounts'])
        for i in range(len(self.containers_list)):
            print(i)
            # print(self.containers_list[i])
            print(self.containers_list[i]['Names'])

    def get_limit(self):
        r = requests.get("http://" + self.nas_ip + ":5055/docker/api/endpoints/" + str(self.endpointsId) +
                         "/dockerhub/0", headers=self.header)
        info = r.json()
        print(info)
        print(info['remaining'])
        if info['remaining'] > 0:
            return True
        else:
            return False

    def get_new_image(self, num):
        image_name = self.containers_list[num]['Image']
        print(image_name)
        r = requests.post("http://" + self.nas_ip + ":5055/docker/api/endpoints/" + str(self.endpointsId) +
                          "/docker/images/create?fromImage=" + image_name, headers=self.header)
        print(r.text)
        if r.status_code == 200:
            return True
        else:
            return False

    def stop_container(self, num):
        r = requests.post("http://" + self.nas_ip + ":5055/docker/api/endpoints/" + str(self.endpointsId) +
                          "/docker/containers/" + self.containers_list[num]['Id'].replace("sha256:", "") + "/stop",
                          headers=self.header)
        print(r.status_code)
        if r.status_code == 204:
            return True
        else:
            return False

    def start_container(self, num):
        r = requests.post("http://" + self.nas_ip + ":5055/docker/api/endpoints/" + str(self.endpointsId) +
                          "/docker/containers/" + self.containers_list[num]['Id'].replace("sha256:", "") + "/start",
                          headers=self.header)
        print(r.status_code)
        if r.status_code == 204:
            return True
        else:
            return False

    def get_containers_info(self, num):
        print(self.containers_list[num]['Id'].replace("sha256:", ""))
        r = requests.get("http://" + self.nas_ip + ":5055/docker/api/endpoints/" + str(self.endpointsId) +
                         "/docker/containers/" + self.containers_list[num]['Id'].replace("sha256:", "") + "/json"
                         , headers=self.header)
        info = r.json()
        print(info)
        return info

    def create_container(self, container_name):
        r = requests.post("http://" + self.nas_ip + ":5055/docker/api/endpoints/" + str(self.endpointsId) +
                          "/docker/containers/create?name=" + container_name, headers=self.header, json=self.body)
        print(r.text)
        if r.status_code == 200:
            return True
        else:
            return False

    def rename_container(self, num, new_name):
        r = requests.post("http://" + self.nas_ip + ":5055/docker/api/endpoints/" + str(self.endpointsId) +
                          "/docker/containers/" + self.containers_list[num]['Id'].replace("sha256:", "") + "/rename"
                                                                                                           "?name=" +
                          new_name, headers=self.header)
        print(r.text)
        if r.status_code == 204:
            return True
        else:
            return False
