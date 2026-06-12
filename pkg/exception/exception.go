package exception

import "errors"

var InvalidInput = errors.New("Invalid Input")

var InvalidFilePath = errors.New("Filepath outside the permiteed working directory")
