package main

import (
	"github.com/gin-gonic/gin"
)

type Runner struct {
	engine gin.Engine
}

func NewRunner() *Runner {
	engine := *gin.Default()

	return &Runner{
		engine: engine,
	}
}
