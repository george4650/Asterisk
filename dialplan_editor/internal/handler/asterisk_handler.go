package handler

import (
	"context"
	"myapp/internal/apperrors"
	"myapp/internal/usecase"
	"myapp/pkg/jaegerotel"
	"net/http"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/codes"

	"github.com/gin-gonic/gin"
)

type AsteriskHandler struct {
	us usecase.AsteriskUseCases
}

// GetFiles получить файлы из директории.
//
//	@Summary		Получить файлы
//	@Description	Получить файлы.
//	@Tags			GetFiles
//	@Produce		json
//	@Param			query	body	handler.GetFiles.GetFilesRequest	true	"Модификатор запроса"
//	@Success		200		{array}	[]models.File
//	@Router			/get-files [get]
func (h *AsteriskHandler) GetFiles(c *gin.Context) {

	type GetFilesRequest struct {
		Path   string `form:"path"`
		Server int    `form:"server"`
	}

	request := GetFilesRequest{}
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"result":  false,
			"message": err.Error(),
			"path":    request.Path,
			"content": nil,
		})
		return
	}

	_, span := jaegerotel.StartSpan(context.Background(), "/get-files")
	defer span.End()

	filesInfo, path, info, err := h.us.GetFiles(c.Request.Context(), request.Path, request.Server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"result":  false,
			"message": err.Error(),
			"path":    path,
			"content": filesInfo,
		})
		span.SetStatus(codes.Error, err.Error())
		span.End()
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result":  true,
		"message": info,
		"path":    path,
		"content": filesInfo,
	})
}

// CreateDir Создать новую директорию.
//
//	@Summary		Создать новую директорию
//	@Description	Создать новую директорию.
//	@Tags			CreateDir
//	@Produce		json
//	@Param			query	body	handler.CreateDir.CreateDirRequest	true	"Модификатор запроса"
//	@Router			/create-dir [post]
func (h *AsteriskHandler) CreateDir(c *gin.Context) {

	type CreateDirRequest struct {
		Path    string `form:"path"`
		DirName string `form:"dirname" binding:"required"`
	}

	request := CreateDirRequest{}
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"result":  false,
			"message": err.Error(),
		})
		return
	}

	_, span := jaegerotel.StartSpan(context.Background(), "/create-dir")
	defer span.End()

	message, err := h.us.CreateDir(c.Request.Context(), request.Path, request.DirName)
	if err != nil {
		switch err {
		case apperrors.ErrFolderAlreadyExist:
			c.JSON(http.StatusBadRequest, gin.H{
				"result":  false,
				"message": message,
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		case apperrors.ErrDirectoryNotExist:
			c.JSON(http.StatusBadRequest, gin.H{
				"result":  false,
				"message": message,
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"result":  false,
				"message": err.Error(),
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"result":  true,
		"message": message,
	})
}

// UploadFiles Закачка файлов на сервер телефонии.
//
//	@Summary		Добавить файлы на сервер
//	@Description	Добавить файлы на сервер.
//	@Tags			UploadFiles
//	@Produce		json
//	@Param			query	body	handler.UploadFiles.UploadFilesRequest	true	"Модификатор запроса"
//	@Router			/upload-files [post]
func (h *AsteriskHandler) UploadFiles(c *gin.Context) {

	type UploadFilesRequest struct {
		Path        string   `form:"path"`
		ConvertList []string `form:"convert_list"`
		Extension   string   `form:"extension"`
	}

	request := UploadFilesRequest{}
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"result":  false,
			"message": err.Error(),
		})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"result":  false,
			"message": err.Error(),
		})
		return
	}
	files, ok := form.File["file"]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"result":  false,
			"message": "Не выбраны файлы для загрузки",
		})
		return
	}
	log.Info().Msgf("%d - файлов загружено", len(files))

	_, span := jaegerotel.StartSpan(context.Background(), "/upload-files")
	defer span.End()

	message, err := h.us.UploadFiles(c.Request.Context(), files, request.Path, request.ConvertList, request.Extension)
	if err != nil {
		switch err {
		case apperrors.ErrBadExtension:
			log.Error().Err(err)
			c.JSON(http.StatusBadRequest, gin.H{
				"result":  false,
				"message": message,
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return

		case apperrors.ErrDirectoryNotExist:
			c.JSON(http.StatusBadRequest, gin.H{
				"result":  false,
				"message": message,
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		case apperrors.ErrBadFileExstension:
			c.JSON(http.StatusBadRequest, gin.H{
				"result":  false,
				"message": message,
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"result":  false,
				"message": err.Error(),
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"result":  true,
		"message": message,
	})

}

// GetAudio Выгрузить аудиофайл.
//
//	@Summary		Получить аудиофайл
//	@Description	Получить аудиофайл.
//	@Tags			GetAudio
//	@Produce		json
//	@Param			query	body	handler.GetAudio.GetAudioRequest	true	"Модификатор запроса"
//	@Success		200		{array}	[]byte
//	@Router			/get-audio [get]
func (h *AsteriskHandler) GetAudio(c *gin.Context) {

	type GetAudioRequest struct {
		FileName string `form:"file" binding:"required"`
		Path     string `form:"path"`
	}

	request := GetAudioRequest{}
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"result":  false,
			"message": err.Error(),
		})
		return
	}

	_, span := jaegerotel.StartSpan(context.Background(), "/get-audio")
	defer span.End()

	fileData, message, err := h.us.GetAudio(c.Request.Context(), request.FileName, request.Path)
	if err != nil {
		switch err {
		case apperrors.ErrFileNotFound:
			c.JSON(http.StatusBadRequest, gin.H{
				"result":  false,
				"message": message,
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"result":  false,
				"message": err.Error(),
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		}
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+request.FileName)
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", fileData)
}

// GetScript получить содержимое файла скрипта.
//
//	@Summary		Получить скрипт файл
//	@Description	Получить скрипт файл.
//	@Tags			GetScript
//	@Produce		json
//	@Param			query	body	handler.GetScript.GetScriptRequest	true	"Модификатор запроса"
//	@Router			/get-script [get]
func (h *AsteriskHandler) GetScript(c *gin.Context) {

	type GetScriptRequest struct {
		FileName string `form:"file" binding:"required"`
		Path     string `form:"path"`
	}

	request := GetScriptRequest{}
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"result":  false,
			"message": err.Error(),
			"content": "",
		})
		return
	}

	_, span := jaegerotel.StartSpan(context.Background(), "/get-script")
	defer span.End()

	content, message, err := h.us.GetScript(c.Request.Context(), request.FileName, request.Path)
	if err != nil {
		switch err {
		case apperrors.ErrFileNotFound:
			c.JSON(http.StatusBadRequest, gin.H{
				"result":  false,
				"message": message,
				"content": "",
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"result":  false,
				"message": err.Error(),
				"content": "",
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"result":  true,
		"message": "",
		"content": content,
	})
}

// UpdateScript Обновление содержимого файла скрипта
//
//	@Summary		Обновить скрипт файл
//	@Description	Обновить скрипт файл.
//	@Tags			UpdateScript
//	@Produce		json
//	@Param			query	body	handler.UpdateScript.UpdateScriptRequest	true	"Модификатор запроса"
//	@Router			/update-script [post]
func (h *AsteriskHandler) UpdateScript(c *gin.Context) {

	type UpdateScriptRequest struct {
		FileName string `form:"file" binding:"required"`
		Path     string `form:"path"`
		Content  string `form:"content" binding:"required"`
	}

	request := UpdateScriptRequest{}
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"result":  false,
			"message": err.Error(),
		})
		return
	}
	_, span := jaegerotel.StartSpan(context.Background(), "/update-script")
	defer span.End()

	message, err := h.us.UpdateScript(c.Request.Context(), request.FileName, request.Path, request.Content)
	if err != nil {
		switch err {
		case apperrors.ErrFileNotFound:
			c.JSON(http.StatusBadRequest, gin.H{
				"result":  false,
				"message": message,
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"result":  false,
				"message": err.Error(),
			})
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"result":  true,
		"message": "",
	})
}

// Функция для получения корневой директории локально (в задании нет)
func (h *AsteriskHandler) GetRoot(c *gin.Context) {

	err := h.us.GetRoot(c.Request.Context())
	if err != nil {
		log.Error().Err(err)
	}
	c.JSON(http.StatusOK, gin.H{
		"result": true,
	})
	return
}
