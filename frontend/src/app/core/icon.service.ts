import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { ApiResponse } from './compose.service';

export interface IconMap { [name: string]: string; }

const builtInImageIcons: Record<string, string> = {
  '1panel': '1Panel.png', alist: 'Alist.png', audiobookshelf: 'Audiobookshelf.png', bilibili: 'Bilibili.png', bitwarden: 'Bitwarden.png', calibre: 'Calibre.png', clash: 'Clash.png', docker: 'Docker.png', emby: 'Emby.png', firefox: 'Firefox.png', ha: 'Ha.png', halo: 'Halo.png', jellyfin: 'Jellyfin.png', komga: 'Komga.png', lucky: 'Lucky.png', mdcng: 'Mdcng.png', moviepilot: 'Moviepilot.png', mysql: 'Mysql.png', navidrome: 'Navidrome.png', nginx: 'Nginx.png', npm: 'Npm.png', plex: 'Plex.png', qbittorrent: 'Qbittorrent.png', qiandao: 'Qiandao.png', rustdesk: 'Rustdesk.png', siyuan: 'Siyuan.png', speedtest: 'Speedtest.png', tailscale: 'Tailscale.png', transmission: 'Transmission.png', v2ray: 'V2ray.png', verysync: 'Verysync.png', zspace: 'Zspace.png', ddnsgo: 'ddnsgo.png', dockercopilot: 'dockercopilot.png', mediasaber: 'mediaSaber.png', postgresql: 'postgresql.png', qinglong: 'qinglong.png', redis: 'redis.png', ugreen: 'ugreen.png', webssh: 'webssh.png', default: 'defaultIcon.png',
};

const imageAliases: Record<string, string> = {
  jellyfin: 'jellyfin', mysqlserver: 'mysql', postgres: 'postgresql', redisstack: 'redis', nginxproxymanager: 'npm', gluetun: 'clash', homeassistant: 'ha', uptimekuma: 'speedtest',
};

@Injectable({ providedIn: 'root' })
export class IconService {
  private readonly http = inject(HttpClient);
  list(): Observable<ApiResponse<IconMap>> { return this.http.get<ApiResponse<IconMap>>('/api/icons'); }
  upload(imageName: string, file: File, containerName = ''): Observable<ApiResponse<string>> {
    const form = new FormData(); form.append('imageName', imageName); form.append('containerName', containerName); form.append('file', file);
    return this.http.post<ApiResponse<string>>('/api/icons', form);
  }
  remove(imageName: string): Observable<ApiResponse<unknown>> { return this.http.delete<ApiResponse<unknown>>('/api/icons?imageName=' + encodeURIComponent(imageName)); }
  url(path: string): string { return path || 'assets/imageIcons/defaultIcon.png'; }
  actionIcon(name: string): string { return 'assets/icons/' + name + '.svg'; }
  imageIcon(imageName: string): string {
    const key = this.normalize(imageName);
    const file = builtInImageIcons[imageAliases[key] || key] || builtInImageIcons['default'];
    return 'assets/imageIcons/' + file;
  }
  resolve(imageName: string, custom: IconMap): string {
    const key = this.normalize(imageName);
    const customKey = Object.keys(custom).find(name => this.normalize(name) === key);
    return (customKey && custom[customKey]) || this.imageIcon(imageName);
  }
  builtInIcons(): [string, string][] { return Object.entries(builtInImageIcons).filter(([name]) => name !== 'default').map(([name, file]) => [name, 'assets/imageIcons/' + file]); }
  private normalize(imageName: string): string {
    let value = (imageName || '').trim().toLowerCase().split('@')[0];
    const lastSlash = value.lastIndexOf('/');
    if (lastSlash >= 0) value = value.slice(lastSlash + 1);
    const lastColon = value.lastIndexOf(':');
    if (lastColon >= 0) value = value.slice(0, lastColon);
    return value.replace(/[^a-z0-9]+/g, '');
  }
}
