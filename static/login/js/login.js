document.addEventListener('DOMContentLoaded', function () {
    // 选择所有的 .form-control 元素
    const inputs = document.querySelectorAll('.form-control');

    // 为每个 input 添加 input 事件监听器
    inputs.forEach(function (input) {
        // 获取每个 input 对应的提交按钮
        const submitArrow = input.nextElementSibling.nextElementSibling; // 假设按钮紧跟在 label 后面

        function toggleButton() {
            if (input.value) {
                input.classList.add('has-content');
                submitArrow.style.display = 'block';
            } else {
                input.classList.remove('has-content');
                submitArrow.style.display = 'none';
            }
        }

        // 检查初始状态并绑定事件
        toggleButton();
        input.addEventListener('input', toggleButton);
    });
});

/**
 * @typedef {Object} LoginResponseData
 * @property {string} jwt
 * @property {string} msg
 */

document.getElementById('loginForm').addEventListener('submit', function (event) {
    // 阻止表单地默认提交行为
    event.preventDefault();

    // 获取 secretKey 的值
    const secretKey = document.getElementById('secretKey').value;

    // 创建 FormData 对象
    const formData = new FormData();
    formData.append('secretKey', secretKey);

    // 发送请求到 /api/auth
    fetch('/api/auth', {
        method: 'POST',
        body: formData
    })
        .then(response => response.json())
        .then(loginResponse => {
            if (loginResponse.code === 201) {
                // 存储 JWT 到 localStorage
                localStorage.setItem('jwt', loginResponse.data.jwt);
                window.location.href = '/manager';
            } else {
                alert('登录失败，请重试' + loginResponse.msg)
            }

        })
        .catch(error => {
            console.error('Error:', error);
            // 处理错误情况
            alert('出现错误，请重试' + error)
        });
});

function isJwtExpired(jwt) {
    try {
        const payloadBase64 = jwt.split('.')[1];
        const decodedPayload = atob(payloadBase64);
        const payload = JSON.parse(decodedPayload);

        // 获取当前时间的 Unix 时间戳（秒）
        const now = Math.floor(Date.now() / 1000);

        // 检查 exp 字段是否存在且是否已过期
        return payload.exp && now >= payload.exp;

         // JWT 未过期
    } catch (error) {
        console.error('校验 JWT 失败:', error);
        return true; // 如果解析出错，默认为过期
    }
}

document.addEventListener('DOMContentLoaded', () => {
    const jwt = localStorage.getItem('jwt');
    if (!jwt) {
        return;
    }
    const expired = isJwtExpired(jwt);
    if (!expired) {
        // JWT 未过期，跳转到管理页面
        window.location.href = '/manager';
    }
});


