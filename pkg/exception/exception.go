package exception

import "errors"

var InvalidInput = errors.New("invalid Input")

var InvalidFilePath = errors.New("filepath outside the permitted working directory")
