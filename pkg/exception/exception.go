package exception

import "errors"

var InvalidInput = errors.New("invalid Input")

var InvalidFilePath = errors.New("Filepath outside the permitted working directory")
