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
    print(container_list)
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
    p = {"all": "true"}
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
        return JsonResponse({"status": "success"})
    else:
        return JsonResponse({"status": "failed"})


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
        return JsonResponse({"status": "success"})
    else:
        return JsonResponse({"status": "failed"})


def get_containers_info(self, num):
    print(self.containers_list[num]['Id'].replace("sha256:", ""))
    r = requests.get("http://" + self.nas_ip + ":5055/docker/api/endpoints/" + str(self.endpointsId) +
                     "/docker/containers/" + self.containers_list[num]['Id'].replace("sha256:", "") + "/json"
                     , headers=self.header)
    info = r.json()
    print(info)
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
        return JsonResponse({"status": "success"})
    else:
        return JsonResponse({"status": "failed"})


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

    def get_docker_info(self):
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
