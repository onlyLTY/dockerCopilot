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
        .then(data => {
            // 存储 JWT 到 localStorage
            localStorage.setItem('jwt', data.token);

            // 处理响应数据
        })
        .catch(error => {
            console.error('Error:', error);
            // 处理错误情况
        });
});

