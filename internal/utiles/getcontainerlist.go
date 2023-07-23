package utiles

import (
	"encoding/json"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/svc"
	"github.com/onlyLTY/oneKeyUpdate/v2/internal/types"
	"net/http"
)

func GetContainerList(ctx *svc.ServiceContext) ([]types.Container, error) {
	containerlistdata := []types.Container{}
	params := map[string]string{
		"all": "true",
	}
	jwt, endpointsId, err := GetNewJwt(ctx)
	if err != nil {
		return containerlistdata, err
	}
	url := domain + "/api/endpoints/" + endpointsId + "/docker/containers/json"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return containerlistdata, err
	}
	req.Header.Add("Authorization", jwt)
	query := req.URL.Query()
	for k, v := range params {
		query.Add(k, v)
	}
	req.URL.RawQuery = query.Encode()

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return containerlistdata, err
	}
	defer response.Body.Close()

	err = json.NewDecoder(response.Body).Decode(&containerlistdata)
	if err != nil {
		return containerlistdata, err
	}
	return containerlistdata, nil
}
func CheckImageUpdate(ctx *svc.ServiceContext, containerlistdata []types.Container) []types.Container {
	for i, v := range containerlistdata {
		if _, ok := ctx.HubImageInfo.Data[v.ImageID]; ok {
			if ctx.HubImageInfo.Data[v.ImageID].NeedUpdate {
				containerlistdata[i].Update = true
			}
		}
	}
	return containerlistdata
}
