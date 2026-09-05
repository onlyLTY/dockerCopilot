import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ApiResponse } from './compose.service';

export interface VersionInfo {
  version: string;
  buildDate: string;
}

export interface HealthInfo {
  status: string;
  version: string;
  docker: string;
}

@Injectable({ providedIn: 'root' })
export class VersionService {
  private readonly http = inject(HttpClient);
  local(): Observable<ApiResponse<VersionInfo>> {
    return this.http.get<ApiResponse<VersionInfo>>('/api/version?type=local');
  }
  /** 检查远端是否有新版本；code=200 时 data.remoteVersion 为远端版本号（与本地相同即无更新） */
  remote(): Observable<ApiResponse<{ remoteVersion: string }>> {
    return this.http.get<ApiResponse<{ remoteVersion: string }>>(
      '/api/version?type=remote',
    );
  }
  /** 下载并落盘更新包；成功后后端约 10 秒自动退出，由容器重启拉起新版本 */
  updateProgram(): Observable<ApiResponse<unknown>> {
    return this.http.put<ApiResponse<unknown>>('/api/program', {});
  }
  /** 不鉴权的存活探测，用于自更新后等待服务恢复；响应头/体含当前版本号 */
  health(): Observable<HealthInfo> {
    return this.http.get<HealthInfo>('/healthz');
  }
}
