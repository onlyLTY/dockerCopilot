import hashlib
import logging
import os

import requests
from django.http import HttpResponse
from django.shortcuts import redirect


class VerifyDeviceMiddleware:
    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        # 在这里检查cookie
        device_verified = request.COOKIES.get('device_verified')
        if not device_verified and request.path.startswith('/containersManager/'):
            # 如果设备未验证且请求的是 /containersManager/ 下的 URL，则重定向到登录页面
            return redirect('login:verification_page')
        if not device_verified and request.path.startswith('/imagesManager/'):
            # 如果设备未验证且请求的是 /imagesManager/ 下的 URL，则重定向到登录页面
            return redirect('login:verification_page')
        # 如果设备已验证或不需要验证，则正常处理请求
        response = self.get_response(request)
        return response


class GlobalVariablesMiddleware:
    def __init__(self, get_response):
        self.get_response = get_response

    def __call__(self, request):
        if os.environ.get('account') is None:
            return HttpResponse("请设置账号", content_type="text/plain; charset=utf-8")
        if request.session.get('jwt') is None:
            get_new_jwt(request)
        if check_jwt_valid(request) is False:
            get_new_jwt(request)
        if request.session.get('endpointsId') is None:
            get_endpoints_id(request)
        response = self.get_response(request)
        return response


def check_jwt_valid(request):
    jwt = request.session['jwt']
    header = {
        "Authorization": jwt
    }
    r = requests.get("http://127.0.0.1:9123/api/endpoints",
                     headers=header)
    if 200 == r.status_code:
        return True
    else:
        return False


def get_new_jwt(request):
    body = {
        "username": "admin",
        "password": hashlib.md5(os.environ.get('account').encode('utf-8')).hexdigest()
    }
    r = requests.post("http://127.0.0.1:9123/api/auth", json=body)
    try:
        request.session['jwt'] = r.json()["jwt"]
    except KeyError:
        logging.error("获取jwt失败")
        return HttpResponse("请检查环境变量中account是否正确设置", content_type="text/plain; charset=utf-8")


def get_endpoints_id(request):
    jwt = request.session['jwt']
    print(jwt)
    header = {
        "Authorization": jwt
    }
    r = requests.get("http://127.0.0.1:9123/api/endpoints",
                     headers=header)
    info = r.json()
    request.session['endpointsId'] = str(info[0]['Id'])
