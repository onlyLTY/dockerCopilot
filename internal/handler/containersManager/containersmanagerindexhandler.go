package containersManager

import (
	"github.com/flosch/pongo2"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/logic/containersManager"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"net/http"
)

func ContainersManagerIndexHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := containersManager.NewContainersManagerIndexLogic(r.Context(), svcCtx)
		list, err := l.ContainersManagerIndex()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			t, err := svcCtx.Template.FromFile("templates/containersManager/containersManager.html")
			if err != nil {
				logx.Error(err)
			}
			execute, err := t.ExecuteBytes(pongo2.Context{"container_list": list})
			if err != nil {
				logx.Error(err)
			}
			w.Write(execute)
			httpx.Ok(w)
		}
	}
}

//func checkUpdate(containerList []types.Container) {
//	imageUpdateInfo := ImageInfo.objects.all()
//	for i, container := range containerList {
//		containerImageID := strings.Split(container.ImageID, ":")[1]
//		image := imageUpdateInfo.filter(image_id=containerImageID)
//		if image.exists() {
//			if image.get(image_id=containerImageID).remoteLastUpdatedTime > image.get(image_id=containerImageID).localCreationTime {
//				containerList[i].Update = true
//			} else {
//				containerList[i].Update = false
//			}
//		} else {
//			containerList[i].Update = false
//		}
//	}
//}
