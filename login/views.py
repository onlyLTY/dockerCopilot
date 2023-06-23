import os
from django.shortcuts import render, redirect


def check_device(request):
    # 检查是否存在指定的cookie
    if 'device_verified' in request.COOKIES:
        # 已验证，可以访问内容
        return redirect('containers_manager:containersManager')
    else:
        # 未验证，重定向到验证页面
        return redirect('login:verification_page')


def verification_page(request):
    if request.method == 'POST':
        # 验证设备
        # ...
        # 验证成功，重定向到主页
        secret_key = os.environ.get('SECRET_KEY')
        print(secret_key)
        if request.POST.get('secret_key') == secret_key:
            print(request.POST.get('secret_key'))
            response = redirect('containers_manager:containersManager')
            response.set_cookie('device_verified', 'true', max_age=365 * 24 * 60 * 60)  # 设置1年有效期
            return response
    # 渲染验证页面
    response = render(request, 'login/login.html')
    return response
