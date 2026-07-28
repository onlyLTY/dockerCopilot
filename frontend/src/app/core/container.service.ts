import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ApiResponse } from './compose.service';
export interface ContainerRow { id: string; name: string; status: string; usingImage: string; createTime: string; runningTime: string; haveUpdate: boolean; }
@Injectable({ providedIn: 'root' }) export class ContainerService {
 private readonly http=inject(HttpClient); list():Observable<ApiResponse<ContainerRow[]>>{return this.http.get<ApiResponse<ContainerRow[]>>('/api/containers');}
 start(id:string){return this.http.post<ApiResponse<unknown>>('/api/container/'+encodeURIComponent(id)+'/start',{});}
 stop(id:string){return this.http.post<ApiResponse<unknown>>('/api/container/'+encodeURIComponent(id)+'/stop',{});}
 restart(id:string){return this.http.post<ApiResponse<unknown>>('/api/container/'+encodeURIComponent(id)+'/restart',{});}
 update(id:string){return this.http.post<ApiResponse<unknown>>('/api/container/'+encodeURIComponent(id)+'/update',{});}
}
