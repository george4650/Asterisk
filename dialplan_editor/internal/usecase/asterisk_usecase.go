package usecase

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math/rand"
	"mime/multipart"
	"myapp/internal/apperrors"
	"myapp/internal/models"
	"os"
	"strings"
	"time"
	"unsafe"

	"github.com/pkg/sftp"
	"github.com/povsister/scp"
	"github.com/rs/zerolog/log"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
	"golang.org/x/crypto/ssh"
)

type AsteriskUseCases struct {
	sshConn *ssh.Client
}

func NewAsteriskUseCases(sshConn *ssh.Client) *AsteriskUseCases {
	return &AsteriskUseCases{
		sshConn: sshConn,
	}
}

var (
	rootDir = "/home/scripts"
)

func (us *AsteriskUseCases) GetFiles(ctx context.Context, path string, server int) ([]models.File, string, string, error) {

	var dirPath, message string

	// Directory path
	if path == "" || path == "/" {
		dirPath = fmt.Sprintf("%s", rootDir)
	} else {
		dirPath = fmt.Sprintf("%s/%s", rootDir, path)
	}

	client, err := sftp.NewClient(us.sshConn)
	if err != nil {
		return nil, dirPath, "", err
	}
	defer client.Close()

	if _, err := client.Stat(dirPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Info().Msgf("Указанной директории не существует: %s", dirPath)
			message = fmt.Sprintf("Указанной директории не существует: %s.\nИнформация о корневой директории.", dirPath)
			dirPath = rootDir
		}
	}

	entries, err := client.ReadDir(dirPath)
	if err != nil {
		return nil, dirPath, "", fmt.Errorf("AsteriskUseCases - GetFiles - client.ReadDir: %v", err)
	}
	var filesInfo []models.File
	for _, entry := range entries {
		filesInfo = append(filesInfo, us.getDirInfo(entry))
	}

	return filesInfo, dirPath, message, nil
}

func (us *AsteriskUseCases) getDirInfo(file fs.FileInfo) models.File {

	var fileInfo models.File
	fileInfo.Type = "f"

	if file.IsDir() {
		fileInfo.Type = "d"
	}
	fileInfo.Changed = file.ModTime().Format("2006-01-02 15:04:05")
	fileInfo.Name = file.Name()

	size := file.Size()
	if size < 1024 {
		fileInfo.Size = fmt.Sprintf("%dВ", size)
	} else if size < 1048576 && size >= 1024 {
		fileInfo.Size = fmt.Sprintf("%dК", size/1024)
	} else if size < 1073741824 && size >= 1048576 {
		fileInfo.Size = fmt.Sprintf("%dМ", size/1024/1024)
	} else {
		fileInfo.Size = fmt.Sprintf("%dГ", size/1024/1024/1024)
	}

	return fileInfo
}

func (us *AsteriskUseCases) CreateDir(ctx context.Context, path, dirName string) (string, error) {
	var dirPath, message string

	// Directory path
	if path == "" || path == "/" {
		dirPath = fmt.Sprintf("%s/%s", rootDir, dirName)
	} else {
		dirPath = fmt.Sprintf("%s/%s/%s", rootDir, path, dirName)
	}
	log.Info().Msgf("dirPath: %s", dirPath)

	client, err := sftp.NewClient(us.sshConn)
	if err != nil {
		return "", fmt.Errorf("AsteriskUseCases - CreateDir - sftp.NewClient: %v", err)
	}
	defer client.Close()

	// Проверяем существует ли директория заданная пользователем
	_, err = client.Stat(fmt.Sprintf("%s/%s", rootDir, path))
	if err == nil {
		// Проверяем существует ли папка в уже заданной директории пользователем
		_, err = client.Stat(dirPath)

		// Если папка в указанной пользователем директории существует, то возвращаем ошибку
		if err == nil {
			log.Info().Msgf("Папка - %s в указанной директории уже существует: %s", dirName, dirPath)
			return fmt.Sprintf("Папка - %s в указанной директории уже существует: %s", dirName, dirPath), apperrors.ErrFolderAlreadyExist
		} else if err == os.ErrNotExist {
			// Если папки в заданной директории не существует, то создаём её
			if err := client.Mkdir(dirPath); err != nil {
				return "", fmt.Errorf("AsteriskUseCases - CreateDir - client.Mkdir: %v", err)
			}
			// Если папка успешно создана, то ничего не возвращаем :)
			return "", nil
		}

	} else if err != nil {

		// Если директории нет
		if errors.Is(err, os.ErrNotExist) {
			log.Info().Msgf("Указанной директории не существует: %s", fmt.Sprintf("%s/%s", rootDir, path))

			message = fmt.Sprintf("Указанной директории не существует: %s. Папка - %s создана в корневой директории.", fmt.Sprintf("%s/%s", rootDir, path), dirName)

			// Проверяем есть ли папка с таким названием в корневой директории
			_, err = client.Stat(fmt.Sprintf("%s/%s", rootDir, dirName))

			// Папка в корне уже есть
			if err == nil {
				// Директории не существует + папка в корне уже есть
				log.Info().Msg("Ошибка при создании папки: директории не существует + папка с таким названием в корне уже есть")
				return "Директории не существует + не удалось создать папку в корне: папка с таким названием уже есть", apperrors.ErrDirectoryNotExist
			} else if err == os.ErrNotExist {

				// Если папки в корне не существует - создаём её
				if err := client.Mkdir(fmt.Sprintf("%s/%s/", rootDir, dirName)); err != nil {
					return "", fmt.Errorf("AsteriskUseCases - CreateDir - client.Mkdir: %v", err)
				}
				return message, nil
			}
		}
	}

	return "", errors.New("Внутренняя ошибка сервера")
}

func (us *AsteriskUseCases) UploadFiles(ctx context.Context, files []*multipart.FileHeader, path string, convertList []string, extension string) (string, error) {

	var (
		audioFilesDir       = "./tmp/audioFiles"
		resultAudioFilesDir = "./tmp/resultAudioFiles"
		dirPath             string
	)

	if extension == "" {
		extension = "wav"
	} else if extension != "wav" && extension != "raw" {
		log.Info().Msgf("Расширение файла .%s не поддерживется", extension)
		return fmt.Sprintf("Расширение файла .%s не поддерживется", extension), apperrors.ErrBadExtension
	}

	// Directory path
	if path == "" || path == "/" {
		dirPath = fmt.Sprintf("%s", rootDir)
	} else {
		dirPath = fmt.Sprintf("%s/%s", rootDir, path)
	}

	client, err := sftp.NewClient(us.sshConn)
	if err != nil {
		return "", err
	}
	if _, err := client.Stat(dirPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Sprintf("Директории %s не существует", dirPath), apperrors.ErrDirectoryNotExist
		}
	}

	// Проверяем наличие папок для сохранения аудиофайлов локально
	if _, err := os.Stat(audioFilesDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err = os.Mkdir(audioFilesDir, os.FileMode(0522)); err != nil {
				return "", fmt.Errorf("AsteriskUseCases - UploadFiles - os.Mkdir: %v", err)
			}
		}
	}
	if _, err := os.Stat(resultAudioFilesDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err = os.Mkdir(resultAudioFilesDir, os.FileMode(0522)); err != nil {
				return "", fmt.Errorf("AsteriskUseCases - UploadFiles - os.Mkdir: %v", err)
			}
		}
	}

	for _, file := range files {

		// Транслитирируем файлы с названиями на русском языке
		fileName := translateFileName(file.Filename)

		fileNameSplit := strings.Split(fileName, ".")

		f, err := file.Open()
		defer f.Close()
		if err != nil {
			return "", err
		}

		var needConvert bool
		for _, name := range convertList {
			if name == file.Filename {
				needConvert = true
			}
		}

		if needConvert {

			if fileNameSplit[len(fileNameSplit)-1] != "mp3" && fileNameSplit[len(fileNameSplit)-1] != "ogg" && fileNameSplit[len(fileNameSplit)-1] != "wav" {
				log.Info().Msgf("Формат файла .%s не поддерживается. Невозможно конвертировать файл - %s", fileNameSplit[len(fileNameSplit)-1], file.Filename)
				return fmt.Sprintf("Формат файла .%s не поддерживается. Невозможно конвертировать файл - %s", fileNameSplit[len(fileNameSplit)-1], file.Filename), apperrors.ErrBadFileExstension
			}
			if fileNameSplit[len(fileNameSplit)-1] == "raw" {
				log.Info().Msgf("Формат файла .raw не поддерживается. Невозможно конвертировать файл - %s", file.Filename)
				return fmt.Sprintf("Формат файла .raw не поддерживается. Невозможно конвертировать файл - %s", file.Filename), apperrors.ErrBadFileExstension
			}

			// Создаём локально аудиофайл
			randomName := generateRandomFilename()
			localFileNameBeforeConvert := fmt.Sprintf("%s.%s", randomName, fileNameSplit[len(fileNameSplit)-1]) // Название файла + расширение до конвертации
			localFileNameAfterConvert := fmt.Sprintf("%s.%s", randomName, extension)                            // Название файла + расширение после конвертации
			localFile, err := os.Create(fmt.Sprintf("%s/%s", audioFilesDir, localFileNameBeforeConvert))
			defer os.Remove(localFile.Name())
			defer localFile.Close()

			if err != nil {
				return "", fmt.Errorf("AsteriskUseCases - UploadFiles - os.Create: %v", err)
			}

			// Записываем данные
			_, err = localFile.ReadFrom(f)
			if err != nil {
				return "", fmt.Errorf("AsteriskUseCases - UploadFiles - localFile.ReadFrom: %v", err)
			}

			// Новое имя файла с выбранным расширением для выгрузки на уд.сервер
			var remoteFileName string
			for i := 0; i < len(fileNameSplit)-1; i++ {
				remoteFileName += fileNameSplit[i] + "."
			}
			remoteFileName += extension

			// Проверяем желаемый формат файла для конвертации
			if extension == "wav" {

				//Проверяем текущее расширение файла
				if fileNameSplit[len(fileNameSplit)-1] == "mp3" || fileNameSplit[len(fileNameSplit)-1] == "ogg" || fileNameSplit[len(fileNameSplit)-1] == "wav" {

					// ffmpeg -i source_file -acodec pcm_s16le -ar 8000 -vol 550 -ac 1 -y output_file							рабочий пример: //ffmpeg -i ./tmp/audioFiles/gffgfg228.mp3 -ac 1 -acodec pcm_s16le -ar 8000 -af volume=7 ./tmp/resultAudioFiles/gffgfg228.wav -y
					i := ffmpeg_go.Input(localFile.Name())
					err = ffmpeg_go.Output([]*ffmpeg_go.Stream{i}, fmt.Sprintf("%s/%s", resultAudioFilesDir, localFileNameAfterConvert), ffmpeg_go.KwArgs{"acodec": "pcm_s16le", "ar": "8000", "ac": "1"}).OverWriteOutput().Run()
					if err != nil {
						return "", fmt.Errorf("AsteriskUseCases - UploadFiles - ffmpeg_go.Output: %v", err)
					}
					err := copyFileToRemote(client, resultAudioFilesDir, localFileNameAfterConvert, dirPath, remoteFileName)
					if err != nil {
						return "", fmt.Errorf("AsteriskUseCases - UploadFiles - copyFileToRemote: %v", err)
					}
					os.Remove(fmt.Sprintf("%s/%s", resultAudioFilesDir, localFileNameAfterConvert))

					//	// Если текущее расширение файла "raw"
					//} else if fileNameSplit[len(fileNameSplit)-1] == "raw" {
					//	// ffmpeg -f s16le -ar 8000 -ac 1 -i output4.raw test22.wav
					//	return "", fmt.Errorf("Формат файла .raw не поддерживается. Невозможно конвертировать файл - %s", file.Filename)
				}

				// Если выходной формат файла "raw"
			} else if extension == "raw" {

				// ffmpeg -i test.mp3 -f s16le -ar 8000 -ac 1 output4.raw
				i := ffmpeg_go.Input(localFile.Name())
				err = ffmpeg_go.Output([]*ffmpeg_go.Stream{i}, fmt.Sprintf("%s/%s", resultAudioFilesDir, localFileNameAfterConvert), ffmpeg_go.KwArgs{"f": "s16le", "ar": "8k", "ac": "1"}).OverWriteOutput().Run()
				if err != nil {
					return "", err
				}
				err := copyFileToRemote(client, resultAudioFilesDir, localFileNameAfterConvert, dirPath, remoteFileName)
				if err != nil {
					return "", err
				}
				os.Remove(fmt.Sprintf("%s/%s", resultAudioFilesDir, localFileNameAfterConvert))
			}
		} else {
			// Если файл конвертировать не требуется, то просто сохраняем его на уд. сервер
			remoteFile, _ := client.Create(fmt.Sprintf("%s/%s", dirPath, fileName))
			defer remoteFile.Close()

			_, err = remoteFile.ReadFrom(f)
			if err != nil {
				return "", fmt.Errorf("AsteriskUseCases - UploadFiles - remoteFile.ReadFrom: %v", err)
			}
			log.Print("Название на уд. серевере", fmt.Sprintf("%s/%s", dirPath, fileName))
		}
	}
	return "", nil
}

// Скопировать файл на удалённый сервер
func copyFileToRemote(client *sftp.Client, localDir, localFileName, remoteDir, remoteFileName string) error {

	f, err := os.Open(fmt.Sprintf("%s/%s", localDir, localFileName))
	if err != nil {
		return errors.New("Внутренняя ошибка сервера")
	}
	defer f.Close()

	log.Print("Результирующий файл - ", f.Name())

	// Сохраняем конвертированный файл на уд. сервер
	log.Print("Название на уд. серевере", fmt.Sprintf("%s/%s", remoteDir, remoteFileName))
	remoteFile, _ := client.Create(fmt.Sprintf("%s/%s", remoteDir, remoteFileName))
	defer remoteFile.Close()
	_, err = remoteFile.ReadFrom(f)
	if err != nil {
		return fmt.Errorf("Не удалось записать файл: %v", err)
	}
	return nil
}

func generateRandomFilename() string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	length := 10
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}

func translateFileName(name string) string {

	symbols := map[string]string{"!": "_", "@": "_", "\"": "_", "#": "_", "№": "N", "$": "S", "%": "_", "^": "_", "{": "_", "}": "_", "(": "_", ")": "_", "'": "_", "~": "_", "`": "_", " ": "_", "«": "_", "+": "_", "=": "_", "[": "_", "]": "_"}

	result := make([]string, len(name))

	for i, symbol := range name {
		if val, ok := symbols[string(symbol)]; ok {
			result[i] = val
			continue
		}
		result[i] = string(symbol)
	}

	log.Print(unsafe.Sizeof(result))

	return strings.Join(result, "")
}

func (us *AsteriskUseCases) GetAudio(ctx context.Context, fileName, path string) ([]byte, string, error) {

	// Directory path
	var filePath string

	if path == "" || path == "/" {
		filePath = fmt.Sprintf("%s/%s", rootDir, fileName)
	} else {
		filePath = fmt.Sprintf("%s/%s/%s", rootDir, path, fileName)
	}

	client, err := sftp.NewClient(us.sshConn)
	if err != nil {
		return nil, "", fmt.Errorf("AsteriskUseCases - GetAudio - sftp.NewClient: %v", err)
	}

	if f, err := client.Stat(filePath); err == nil {
		file, err := client.Open(filePath)
		if err != nil {
			return nil, "", fmt.Errorf("AsteriskUseCases - GetAudio - client.Open: %v", err)
		}
		defer file.Close()

		var size int64 = f.Size()
		bytes := make([]byte, size)

		bufr := bufio.NewReader(file)
		_, err = bufr.Read(bytes)

		return bytes, "", nil

	} else if err != nil {
		if err == os.ErrNotExist {
			return nil, fmt.Sprintf("Файла %s в директории %s не существует", fileName, fmt.Sprintf("%s/%s", rootDir, path)), apperrors.ErrFileNotFound
		}
		return nil, "", err
	}

	return nil, "", errors.New("Внутренняя ошибка сервера")
}

func (us *AsteriskUseCases) GetScript(ctx context.Context, fileName, path string) (string, string, error) {

	var filePath string

	if path == "" || path == "/" {
		filePath = fmt.Sprintf("%s/%s", rootDir, fileName)
	} else {
		filePath = fmt.Sprintf("%s/%s/%s", rootDir, path, fileName)
	}

	client, err := sftp.NewClient(us.sshConn)
	if err != nil {
		return "", "", err
	}

	if _, err := client.Stat(filePath); err == nil {
		file, _ := client.Open(filePath)
		if err != nil {
			return "", "", fmt.Errorf("AsteriskUseCases - GetScript - client.Open: %v", err)
		}
		defer file.Close()

		var script strings.Builder

		rd := bufio.NewReader(file)
		log.Print("rd  ", rd)
		for {
			line, err := rd.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				return "", "", fmt.Errorf("AsteriskUseCases - GetScript - rd.ReadString: %v", err)
			}
			script.WriteString(line)
		}
		log.Info().Msgf("%v", script.String())

		return script.String(), "", nil

	} else if err != nil {
		if err == os.ErrNotExist {
			return "", fmt.Sprintf("Файла %s в директории %s не существует", fileName, fmt.Sprintf("%s/%s", rootDir, path)), apperrors.ErrFileNotFound
		}
		return "", "", fmt.Errorf("AsteriskUseCases - GetScript - client.Stat: %v", err)
	}
	return "", "", errors.New("Внутренняя ошибка сервера")
}

func (us *AsteriskUseCases) UpdateScript(ctx context.Context, fileName, path, content string) (string, error) {

	// Directory path
	var filePath string

	if path == "" || path == "/" {
		filePath = fmt.Sprintf("%s/%s", rootDir, fileName)
	} else {
		filePath = fmt.Sprintf("%s/%s/%s", rootDir, path, fileName)
	}

	client, err := sftp.NewClient(us.sshConn)
	if err != nil {
		return "", fmt.Errorf("AsteriskUseCases - UpdateScript - sftp.NewClient: %v", err)
	}

	// Проверяем наличие файла в уд. директории
	if _, err := client.Stat(filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Sprintf("Файла %s не существует в директории %s", fileName, fmt.Sprintf("%s/%s", rootDir, path)), apperrors.ErrFileNotFound
		}
	}

	f, err := client.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("AsteriskUseCases - UpdateScript - client.Create: %v", err)
	}
	_, err = f.Write([]byte(content))
	if err != nil {
		return "", fmt.Errorf("AsteriskUseCases - UpdateScript - f.Write: %v", err)
	}

	return "", nil
}

// ... Для получения папки из корня локально (нет в задании)
func (us *AsteriskUseCases) GetRoot(ctx context.Context) error {
	if _, err := os.Stat("./tmp/rootDir"); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err = os.Mkdir("./tmp/rootDir", os.FileMode(0522)); err != nil {
				return fmt.Errorf("AsteriskUseCases - GetRoot - os.Create: %v", err)
			}
		}
	}
	scpClient, err := scp.NewClientFromExistingSSH(us.sshConn, &scp.ClientOption{})

	scpClient.CopyDirFromRemote(rootDir, "./tmp/rootDir", &scp.DirTransferOption{})

	if err != nil {
		return err
	}

	return nil
}
