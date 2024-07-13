# -*- coding: utf-8 -*-

import os
import time
from typing import TypedDict, Generator, Any

from webdav3.client import Client
from webdav3.exceptions import *


class File(TypedDict):
    dir: str
    name: str
    ext: str


class StrmGenerator:
    def __init__(self, webdav_url: str, username: str, password: str, save_dir: str, remote_path: str,
                 retry_times: int = 3):
        self.options = {
            'webdav_hostname': webdav_url,
            'webdav_login': username,
            'webdav_password': password
        }
        self.client = Client(self.options)
        self.save_dir = save_dir
        self.remote_path = remote_path
        q = 0
        while q < retry_times:
            try:
                if not self.client.check(remote_path):
                    raise ConnectionException("连接失败")
                break
            except ConnectionException:
                q += 1
        else:
            raise ConnectionException("多次重连失败")

    def list_files(self):
        def get_files(dir: str = ''):
            for file in self.client.list(dir)[1:]:
                if file.endswith('/'):
                    for child_file in get_files(dir + file):  # type: File
                        yield child_file
                else:
                    t = file.split('.')
                    name, ext = t[0], t[-1]
                    if len(t) == 1:
                        ext = ''
                    else:
                        name = '.'.join(t[:-1])
                    yield File(dir=dir, name=name, ext=ext.lower())

        for file in get_files(self.remote_path):
            yield file

    def generate_strm_by_files(self, webdav_files: Generator[File, Any, None], download_ext: tuple[str],
                               strm_ext: tuple[str]):
        for file in webdav_files:
            absolute_path = os.path.normpath(f'{file["dir"]}{file["name"]}.{file["ext"]}').replace('\\', '/')
            relative_path = os.path.relpath(absolute_path, self.remote_path)
            save_path = os.path.join(self.save_dir, relative_path).replace('\\', '/')
            save_dir = os.path.dirname(save_path)
            if not os.path.exists(save_dir):
                os.makedirs(save_dir)
            if file['ext'] in download_ext:
                if os.path.exists(save_path):
                    continue
                self.client.download_sync(absolute_path, save_path)
            elif file['ext'] in strm_ext:
                if os.path.exists(os.path.join(save_dir, f'{file["name"]}.strm')):
                    continue
                with open(os.path.join(save_dir, f'{file["name"]}.strm'), 'w',
                          encoding='utf-8') as f:
                    f.write(f'{self.options['webdav_hostname']}{absolute_path.replace('/dav', '/d', 1)}')
            time.sleep(0.1)

    def generate(self, download_ext: tuple[str] = ('jpg', 'png', 'webm', 'gif', 'srt', 'ass', 'ssa', 'nfo'),
                 strm_ext: tuple[str] = ('mp4', 'mkv', 'flv', 'avi')):
        self.generate_strm_by_files(self.list_files(), download_ext, strm_ext)


if __name__ == '__main__':
    webdav_url = 'yourdomain:port'  # alist 地址
    remote_path = '/dav/path/'
    save_dir = 'Z:/OpenWrt/mnt/mmcblk2p4/TV/电影/'  # 输出路径
    username = 'yourusername'  # 用户名
    password = 'yourpassword'  # 密码
    strm_generator = StrmGenerator(webdav_url=webdav_url, remote_path=remote_path, username=username, password=password,
                                   save_dir=save_dir)
    strm_generator.generate()
