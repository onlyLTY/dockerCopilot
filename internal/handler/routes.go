// Code generated by goctl. DO NOT EDIT.
package handler

import (
	"net/http"

	Login "github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/handler/Login"
	auth "github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/handler/auth"
	container "github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/handler/container"
	containersManager "github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/handler/containersManager"
	imagesManager "github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/handler/imagesManager"
	progress "github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/handler/progress"
	version "github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/handler/version"
	"github.com/onlyLTY/oneKeyUpdate/UGREEN/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/",
				Handler: webindexHandler(serverCtx),
			},
		},
	)

	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.IndexCheckMiddleware},
			[]rest.Route{
				{
					Method:  http.MethodPost,
					Path:    "/login",
					Handler: Login.DoLoginHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/login",
					Handler: Login.LoginIndexHandler(serverCtx),
				},
			}...,
		),
		rest.WithPrefix("/"),
	)

	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.CookieCheckMiddleware},
			[]rest.Route{
				{
					Method:  http.MethodGet,
					Path:    "/",
					Handler: containersManager.ContainersManagerIndexHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/start_container",
					Handler: containersManager.StartContainerHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/stop_container",
					Handler: containersManager.StopContainerHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/rename_container",
					Handler: containersManager.RenameContainerHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/create_container",
					Handler: containersManager.CreateContainerHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/remove_container",
					Handler: containersManager.RemoveContainerHandler(serverCtx),
				},
			}...,
		),
		rest.WithPrefix("/containersManager"),
	)

	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.CookieCheckMiddleware},
			[]rest.Route{
				{
					Method:  http.MethodGet,
					Path:    "/",
					Handler: imagesManager.ImagesManagerIndexHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/get_new_image",
					Handler: imagesManager.GetNewImageHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/remove_image",
					Handler: imagesManager.RemoveImageHandler(serverCtx),
				},
			}...,
		),
		rest.WithPrefix("/imagesManager"),
	)

	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.CookieCheckMiddleware},
			[]rest.Route{
				{
					Method:  http.MethodGet,
					Path:    "/",
					Handler: version.VersionIndexHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/get_version",
					Handler: version.GetVersionsHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/updateprogram",
					Handler: version.UpdateprogramHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/checkprogramupdate",
					Handler: version.CheckprogramupdateHandler(serverCtx),
				},
			}...,
		),
		rest.WithPrefix("/version"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/auth",
				Handler: auth.LoginHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api"),
	)

	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodGet,
				Path:    "/progress/:taskid",
				Handler: progress.GetProgressHandler(serverCtx),
			},
		},
		rest.WithPrefix("/api"),
	)

	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.BearerTokenCheckMiddleware},
			[]rest.Route{
				{
					Method:  http.MethodGet,
					Path:    "/containers",
					Handler: container.ListHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/container/:id/start",
					Handler: container.StartHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/container/:id/stop",
					Handler: container.StopHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/container/:id/restart",
					Handler: container.RestartHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/container/:id/rename",
					Handler: container.RenameHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/container/:id/update",
					Handler: container.UpdateHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/container/backup",
					Handler: container.BackupHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/container/listBackups",
					Handler: container.ListBackupsHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/container/backups/:filename/restore",
					Handler: container.RestoreHandler(serverCtx),
				},
			}...,
		),
		rest.WithPrefix("/api"),
	)
}
