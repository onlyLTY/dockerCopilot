import os
from django.shortcuts import render, redirect



def check_device(request):
    # 检查是否存在指定的cookie
    if request.session.get('secret_key') is not None:
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
            request.session['secret_key'] = secret_key
            return response
    # 渲染验证页面
    response = render(request, 'login/login.html')
    return response
