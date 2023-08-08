package apperrors

import "errors"

var (
	ErrFolderAlreadyExist = errors.New("folder already exist")
	ErrDirectoryNotExist  = errors.New("directory not exist")

	ErrBadExtension      = errors.New("err bad extension")
	ErrBadFileExstension = errors.New("err bad file extension")

	ErrFileNotFound = errors.New("err file not found")
)
