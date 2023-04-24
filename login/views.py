import hashlib
import requests
from .models import User
from datetime import datetime
from django.shortcuts import render, redirect
from django.template import loader
from django.http import HttpResponse


def index(request):
    try:
        user_pass = User.objects.get(pk=1)
    except User.DoesNotExist:
        current_year = datetime.now()
        context = {'current_year': current_year}
        template = loader.get_template("login/login.html")
        return HttpResponse(template.render(context, request))
    else:
        phone = user_pass.phone
        nas_ip = user_pass.nasIp
        nas_cookie = user_pass.cookie
        header = {
            'Cookie': nas_cookie
        }
        body = {
            "username": "admin",
            "password": hashlib.md5(phone.encode('utf-8')).hexdigest()
        }
        try:
            r = requests.post("http://" + nas_ip + ":5055/docker/api/auth", headers=header, json=body)
            # 处理响应...
        except requests.exceptions.RequestException as e:
            print(e)  # 打印错误消息
            return render(
                request,
                "login/login.html",
                {
                    "error_message": "NAS服务器连接失败，请检查NAS服务器是否开启或者IP地址是否正确",
                },
            )
        else:
            if r.status_code == 200:
                jwt = r.json()
                print("ok")
                return HttpResponse("ok")
            else:
                return render(
                    request,
                    "login/login.html",
                    {
                        "error_message": "请检查您的号码或者更新cookie",
                    },
                )


def check(request):
    if request.method == "POST":
        phone = request.POST.get("phoneNum")
        nas_ip = request.POST.get("ipAddress")
        nas_cookie = request.POST.get("nasCookie")
        header = {
            'Cookie': nas_cookie
        }
        body = {
            "username": "admin",
            "password": hashlib.md5(phone.encode('utf-8')).hexdigest()
        }
        try:
            r = requests.post("http://" + nas_ip + ":5055/docker/api/auth", headers=header, json=body)
            # 处理响应...
        except requests.exceptions.RequestException as e:
            print(e)  # 打印错误消息
            return render(
                request,
                "login/login.html",
                {
                    "error_message": "请检查您的号码、NAS IP地址和Cookie是否正确",
                },
            )
        else:
            if r.status_code == 200:
                jwt = r.json()
                print("ok")
                try:
                    user_pass = User.objects.get(pk=1)
                except User.DoesNotExist:
                    user = User(phone=phone, nasIp=nas_ip, cookie=nas_cookie)
                    user.save()
                else:
                    user_pass.phone = phone
                    user_pass.nasIp = nas_ip
                    user_pass.cookie = nas_cookie
                    user_pass.save()
                finally:
                    return HttpResponse("ok")

            else:
                return render(
                    request,
                    "login/login.html",
                    {
                        "error_message": "请检查您的号码、NAS IP地址和Cookie是否正确",
                    },
                )
    else:
        return redirect("login:index")
