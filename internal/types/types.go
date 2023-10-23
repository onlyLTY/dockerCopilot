// Code generated by goctl. DO NOT EDIT.
package types

type DoLoginReq struct {
	SecretKey string `form:"secret_key,optional"`
}

type LoginReq struct {
	SecretKey string `form:"secretKey,optional"`
}

type LoginResp struct {
	Token string `json:"token"`
}

type GetProgressReq struct {
	TaskId string `path:"taskid"`
}

type StartContainerReq struct {
	Name string `json:"name"`
}

type StopContainerReq struct {
	Name string `json:"name"`
}

type RenameContainerReq struct {
	OldName string `json:"oldName"`
	NewName string `json:"newName"`
}

type CreateContainerReq struct {
	OldName         string `json:"old_name"`
	NewName         string `json:"new_name"`
	ImageNameAndTag string `json:"image_name_and_tag"`
}

type RemoveContainerReq struct {
	Name string `json:"name"`
}

type GetNewImageReq struct {
	ImageNameAndTag string `json:"image_name_and_tag"`
}

type RemoveImageReq struct {
	Force   bool   `json:"force"`
	ImageID string `json:"imageID"`
}

type MsgResp struct {
	Status string `json:"status"`
	Msg    string `json:"msg"`
}

type Resp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type ContainerIdReq struct {
	Id string `path:"id"`
}

type ContainerRestoreReq struct {
	Filename string `path:"filename"`
}

type ContainerRenameReq struct {
	ContainerIdReq
	NewName string `form:"newName"`
}

type ContainerUpdateReq struct {
	ContainerIdReq
	DelOldContainer bool   `form:"delOldContainer"`
	Proxy           string `form:"proxy,optional"`
	ImageNameAndTag string `form:"imageNameAndTag"`
	Name            string `form:"name"`
}

type VersionMsgResp struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
}
