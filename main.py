import os
import sys
import requests
import hashlib
from PySide6.QtUiTools import QUiLoader
from PySide6.QtWidgets import QApplication, QMessageBox
from PySide6.QtCore import QFile, QIODevice, QStringListModel
from PySide6.QtCore import QUrl
from PySide6.QtWebEngineWidgets import QWebEngineView


def resource_path(relative_path):
    """获取程序中所需文件资源的绝对路径"""
    try:
        # PyInstaller创建临时文件夹,将路径存储于_MEIPASS
        base_path = sys._MEIPASS
    except Exception:
        base_path = os.path.abspath(".")

    return os.path.join(base_path, relative_path)


class Login:
    def __init__(self, account, cookie, nas_ip):
        self.nas_ip = nas_ip
        self.account = account
        self.cookie = cookie.strip()
        self.header = {
            'Cookie': self.cookie
        }
        self.body = {
            "username": "admin",
            "password": hashlib.md5(self.account.encode()).hexdigest()
        }

    def login(self):
        print(self.header)
        print(self.body)
        r = requests.post("http://" + self.nas_ip + ":5055/docker/api/auth", headers=self.header, json=self.body)
        jwt = r.json()
        return jwt


class LoginUI:
    def __init__(self):
        # 先导入.ui文件，存在qfile_UIUI。然后关闭
        self.m = None
        qfile_ui = QFile(resource_path("login.ui"))
        qfile_ui.open(QFile.ReadOnly)
        qfile_ui.close()

        # 导入加载的UI类（返回的就是UI界面对应的QWidget窗体对象）
        self.ui = QUiLoader().load(qfile_ui)  # 界面对象
        self.ui.pushButton_login.clicked.connect(self.login_button)
        self.ui.pushButton_exit.clicked.connect(self.exit_button)

    def login_button(self):  # 与手动创建代码不同，这里需要在（）加入self
        print('try to login')
        account = self.ui.lineEdit_account.text()
        cookie = self.ui.lineEdit_cookie.text()
        nas_ip = self.ui.lineEdit_ip.text()
        print(account)
        print(cookie)
        login = Login(account, cookie, nas_ip)
        jwt = login.login()
        # print(jwt)
        self.m = MainwindowUI()
        docker_update = DockerUpdate(account, cookie, jwt, nas_ip, self.m)
        self.m.set_docker_update(docker_update)
        docker_update.get_endpoints_ID()
        docker_update.get_docker_info()
        # 以下为测试函数
        # docker_update.get_limit()
        # docker_update.getNewImage(0)
        # docker_update.stopContainer(0)
        # docker_update.startContainer(0)
        # docker_update.get_containers_info(0)
        # 测试函数终止
        self.m.ui.show()
        self.ui.close()

    @staticmethod
    def exit_button():
        sys.exit(0)


class DockerUpdate:
    account = None
    cookie = None
    jwt = None
    endpointsId = None
    m = None
    containers_list = None

    def __init__(self, account, cookie, jwt, nas_ip, m):
        self.m = m
        self.account = account
        self.cookie = cookie.strip()
        self.jwt = "Bearer " + jwt["jwt"]
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
        # print(info[0]['Id'])
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
            self.m.update_docker_list(self.containers_list[i]['Names'])

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


class MainwindowUI:
    def __init__(self):
        # 先导入.ui文件，存在qfile_UIUI。然后关闭
        qfile_ui = QFile(resource_path("mainwindow.ui"))
        qfile_ui.open(QFile.ReadOnly)
        qfile_ui.close()

        self.docker_update = None
        # 导入加载的UI类（返回的就是UI界面对应的QWidget窗体对象）
        self.ui = QUiLoader().load(qfile_ui)  # 界面对象
        # self.ui.pushButton_login.clicked.connect(self.loginButton)
        # self.ui.pushButton_exit.clicked.connect(self.exitButton)
        self.string_list_model = QStringListModel()
        self.ui.listView.setModel(self.string_list_model)  # 把view和model关联
        # self.string_list_model.dataChanged.connect(self.save)  # 存储所有行的数据
        self.ui.pushButton_update.clicked.connect(self.update_button)
        self.ui.pushButton_refresh.clicked.connect(self.refresh_button)

    def set_docker_update(self, docker_update):
        self.docker_update = docker_update

    def update_docker_list(self, text):
        row = self.string_list_model.rowCount()
        if text:
            self.string_list_model.insertRow(row)
            self.string_list_model.setData(self.string_list_model.index(row), text)
        else:
            print('null')

    def update_button(self):
        print("updateButton")

        # create a message box object
        msg_box = QMessageBox()
        # set the text and icon for the message box
        msg_box.setText("请不要操作。软件提示无响应是正常情况。更新容器考虑到网络问题，可能需要较长时间。完成后会有提示。")
        msg_box.setIcon(QMessageBox.Information)
        # show the message box and wait for the user to close it
        msg_box.exec()

        select_item = self.ui.listView.selectedIndexes()
        print(select_item[0].row())
        info = self.docker_update.get_containers_info(select_item[0].row())
        for i in info['Config']:
            self.docker_update.body[i] = info['Config'][i]
        self.docker_update.body['HostConfig'] = info['HostConfig']
        self.docker_update.body['name'] = info['Name'].lstrip('/')
        self.docker_update.body['NetworkingConfig'] = {}
        self.docker_update.body['NetworkingConfig']['EndpointsConfig'] = {}
        self.docker_update.body['NetworkingConfig']['EndpointsConfig']['bridge'] = \
            info['NetworkSettings']['Networks']['bridge']
        flag1 = self.docker_update.stop_container(select_item[0].row())
        if flag1:
            flag2 = self.docker_update.rename_container(select_item[0].row(), self.docker_update.body['name'] + '-old')
            if flag2:
                flag3 = self.docker_update.create_container(self.docker_update.body['name'])
                if flag3:
                    self.refresh_button()
                    flag4 = self.docker_update.start_container(0)
                    if flag4:
                        msg_box = QMessageBox()
                        msg_box.setText("更新成功")
                        msg_box.setIcon(QMessageBox.Information)
                        msg_box.exec()
                    else:
                        msg_box = QMessageBox()
                        msg_box.setText("更新失败")
                        msg_box.setIcon(QMessageBox.Information)
                        msg_box.exec()
                else:
                    msg_box = QMessageBox()
                    msg_box.setText("更新失败")
                    msg_box.setIcon(QMessageBox.Information)
                    msg_box.exec()
            else:
                msg_box = QMessageBox()
                msg_box.setText("更新失败")
                msg_box.setIcon(QMessageBox.Information)
                msg_box.exec()
        else:
            msg_box = QMessageBox()
            msg_box.setText("更新失败")
            msg_box.setIcon(QMessageBox.Information)
            msg_box.exec()

    def refresh_button(self):
        print("refreshButton")
        self.string_list_model.removeRows(0, self.string_list_model.rowCount())
        self.docker_update.get_docker_info()


if __name__ == "__main__":
    app = QApplication([])
    loginUi = LoginUI()
    loginUi.ui.show()
    app.exec()
