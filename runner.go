package main

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type Runner struct {
	engine *gin.Engine
}

func NewRunner() *Runner {
	engine := gin.Default()

	return &Runner{
		engine: engine,
	}
}

func (r *Runner) Start() {
	r.initRoutes()
	var (
		addr = viper.GetString("server.addr")
		port = viper.GetString("server.port")
	)

	if err := r.engine.Run(fmt.Sprintf("%s:%s", addr, port)); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}

type CommandRequest struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Timeout int               `json:"timeout,omitempty"` // 超时时间（秒）
	Env     map[string]string `json:"env,omitempty"`     // 环境变量
}

type CommandResponse struct {
	Success   bool   `json:"success"`
	Output    string `json:"output"`
	Error     string `json:"error,omitempty"`
	ExitCode  int    `json:"exitCode,omitempty"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

func (r *Runner) initRoutes() {
	var router = r.engine.Group("/api")
	// GET方式支持直接通过URL参数执行命令
	router.POST("/executions", runCmd)
}

func runCmd(ctx *gin.Context) {
	var req CommandRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"code": -1, "message": err.Error()})
		return
	}

	// 验证命令安全性
	if !isValidCommand(req.Command) {
		ctx.JSON(400, gin.H{"code": -1, "message": "Invalid or unsafe command"})
		return
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 30 // 默认30秒
	}

	result, err := executeCommand(req.Command, req.Args, timeout, req.Env)
	if err != nil {
		ctx.JSON(500, gin.H{"code": -1, "message": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"code": 0, "message": "success", "data": result})
}

// executeCommand 执行系统命令
func executeCommand(command string, args []string, timeout int, env map[string]string) (*CommandResponse, error) {
	start := time.Now()
	startTimeStr := start.Format(time.RFC3339)

	// 创建命令
	cmd := exec.Command(command, args...)

	// 设置环境变量
	if env != nil && len(env) > 0 {
		envVars := make([]string, 0, len(env))
		for k, v := range env {
			envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = append(cmd.Environ(), envVars...)
	}

	// 获取输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %v", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %v", err)
	}

	// 创建通道来处理输出
	stdOutCh := make(chan string, 1) // 缓冲通道
	stdErrCh := make(chan string, 1) // 缓冲通道
	errCh := make(chan error, 1)     // 缓冲通道

	// 并发读取输出
	go func() {
		out, _ := io.ReadAll(stdout)
		stdOutCh <- string(out)
	}()
	go func() {
		errOut, _ := io.ReadAll(stderr)
		stdErrCh <- string(errOut)
	}()
	go func() {
		err := cmd.Wait()
		errCh <- err
	}()

	// 设置超时
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		// 超时，杀死进程
		if killErr := cmd.Process.Kill(); killErr != nil {
			return nil, fmt.Errorf("failed to kill process after timeout: %v", killErr)
		}
		<-errCh // 等待错误通道完成

		endTimeStr := time.Now().Format(time.RFC3339)
		return &CommandResponse{
			Success:   false,
			Output:    "Command timed out",
			Error:     "Command execution timed out",
			ExitCode:  -1,
			StartTime: startTimeStr,
			EndTime:   endTimeStr,
		}, nil

	case cmdErr := <-errCh:
		// 命令完成
		endTimeStr := time.Now().Format(time.RFC3339)

		// 读取输出（使用带超时的select避免死锁）
		var output, errorOutput string

		// 等待输出，带超时
		timeoutChan := time.After(1 * time.Second)

		// 等待stdout
		select {
		case output = <-stdOutCh:
		case <-timeoutChan:
			output = ""
		}

		// 重置超时
		timeoutChan = time.After(1 * time.Second)

		// 等待stderr
		select {
		case errorOutput = <-stdErrCh:
		case <-timeoutChan:
			errorOutput = ""
		}

		fullOutput := output
		if errorOutput != "" {
			fullOutput += "\n" + errorOutput
		}

		exitCode := 0
		if cmdErr != nil {
			if exitError, ok := cmdErr.(*exec.ExitError); ok {
				exitCode = exitError.ExitCode()
			} else {
				exitCode = -1
			}
		}

		var errorMsg string
		if cmdErr != nil {
			errorMsg = cmdErr.Error()
		}

		return &CommandResponse{
			Success:   cmdErr == nil,
			Output:    fullOutput,
			Error:     errorMsg,
			ExitCode:  exitCode,
			StartTime: startTimeStr,
			EndTime:   endTimeStr,
		}, nil
	}
}

// isValidCommand 验证命令是否安全
func isValidCommand(command string) bool {
	if command == "" {
		return false
	}

	// 检查危险字符
	unsafeChars := []string{";", "&", "|", "$", "`", "\n", "\r"}
	for _, char := range unsafeChars {
		if strings.Contains(command, char) {
			return false
		}
	}

	// 检查危险命令
	unsafeCmds := []string{"rm", "mv", "dd", "mkfs", "shutdown", "reboot", "halt", "poweroff"}
	cmdParts := strings.Fields(command)
	if len(cmdParts) > 0 {
		firstPart := strings.ToLower(cmdParts[0])
		for _, unsafeCmd := range unsafeCmds {
			if firstPart == unsafeCmd {
				return false
			}
		}
	}

	return true
}
