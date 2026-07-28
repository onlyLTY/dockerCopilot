package icons

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/onlyLTY/dockerCopilot/internal/svc"
	"github.com/onlyLTY/dockerCopilot/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UploadHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 解析 Multipart 表单
		err := r.ParseMultipartForm(10 << 20) // 10MB 限制
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("failed to parse form: %v", err))
			return
		}

		// 2. 获取文件和 Key
		file, handler, err := r.FormFile("file")
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("failed to get file: %v", err))
			return
		}
		defer file.Close()

		imageNameKey := r.FormValue("imageName")
		if imageNameKey == "" {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("imageName is required"))
			return
		}

		// 3. 确保目录存在 (防御性编程)
		dataPath := "/data/icon/icons"
		os.MkdirAll(dataPath, 0755)

		// 4. 确定文件名
		filename := handler.Filename
		containerName := r.FormValue("containerName")
		if containerName != "" {
			// 清理容器名称以确保文件名安全
			// 将非法字符替换为下划线
			reg := regexp.MustCompile(`[\\/:*?"<>|]`)
			safeName := reg.ReplaceAllString(containerName, "_")

			// 获取原始文件名的扩展名
			ext := filepath.Ext(filename)
			if ext == "" {
				// 尝试从内容类型检测扩展名，或者如果缺失则默认为 .png？
				// 如果可能，最好保留原始扩展名逻辑，或者不追加任何内容。
			}
			filename = safeName + ext
		}

		dstPath := filepath.Join(dataPath, filename)
		dst, err := os.Create(dstPath)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("failed to create file on server: %v", err))
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("failed to copy file content: %v", err))
			return
		}

		// 5. 更新 imageLogos.js
		jsPath := "/data/icon/imageLogos.js"
		if err := updateImageLogosJS(jsPath, imageNameKey, filename); err != nil {
			httpx.ErrorCtx(r.Context(), w, fmt.Errorf("failed to update config: %v", err))
			return
		}

		httpx.OkJsonCtx(r.Context(), w, types.Resp{
			Code: 200,
			Msg:  "Success",
			Data: filename,
		})
	}
}

func updateImageLogosJS(filePath, imageName, filename string) error {
	// 读取文件
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	content := string(contentBytes)

	// 前端使用的容器路径
	containerPath := fmt.Sprintf("/src/config/image/%s", filename)

	if strings.Contains(content, fmt.Sprintf(`"%s"`, imageName)) {
		// 更新现有行
		re := regexp.MustCompile(fmt.Sprintf(`"%s"\s*:\s*".*"`, regexp.QuoteMeta(imageName)))
		content = re.ReplaceAllString(content, fmt.Sprintf(`"%s": "%s"`, imageName, containerPath))
	} else {
		// 插入新行
		// 查找 `export const customImageLogos = {`
		startIdx := strings.Index(content, "export const customImageLogos = {")
		if startIdx == -1 {
			return fmt.Errorf("invalid config format")
		}
		// 尝试查找右大括号。这里假设它是最后一个右大括号逻辑或者是文件末尾。
		// 一个简单的启发式方法：插入到最后一个 `}` 或 `};` 之前。
		lastBraceIdx := strings.LastIndex(content, "}")
		if lastBraceIdx == -1 || lastBraceIdx < startIdx {
			return fmt.Errorf("invalid config format, no closing brace")
		}

		newLine := fmt.Sprintf(`  "%s": "%s",`, imageName, containerPath)
		// 插入到最后一个大括号之前
		content = content[:lastBraceIdx] + newLine + "\n" + content[lastBraceIdx:]
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}
