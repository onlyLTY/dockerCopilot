package icons

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteLogic {
	return &DeleteLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *DeleteLogic) Delete(imageName string) (*types.Resp, error) {
	repository, err := NormalizeRepository(imageName)
	if err != nil {
		return nil, err
	}
	_, err = withIconConfigLock(func() (bool, error) {
		icons, err := readIcons(iconConfigPath())
		if err != nil {
			return false, err
		}
		removed := false
		files := make([]string, 0)
		for _, key := range matchingKeys(icons, repository) {
			files = append(files, filepath.Base(icons[key]))
			delete(icons, key)
			removed = true
		}
		if !removed {
			return false, nil
		}
		if err := writeIcons(iconConfigPath(), icons); err != nil {
			return false, err
		}
		for _, file := range files {
			if !iconFileReferenced(icons, file) {
				if err := os.Remove(filepath.Join(iconDirectory(), file)); err != nil && !os.IsNotExist(err) {
					return false, fmt.Errorf("删除图标文件失败: %v", err)
				}
			}
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.Resp{Code: 200, Msg: "Success", Data: nil}, nil
}
