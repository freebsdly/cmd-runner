package main

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var log *zap.Logger

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cmd-runner",
	Short: "A cmd runner for ai",
	Long:  `Run script which generate by AI`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Sugar().Debug("log level: %s", viper.GetString("log.level"))

		log.Debug("This is a debug message")
		log.Warn("This is a warning")
		log.Error("This is an error")

		log.Info("Application finished")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		if err != nil {
			return
		}
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	// 全局标志：指定配置文件
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "config.yaml", "config file path")

	// 定义命令行 flag（带默认值）
	rootCmd.Flags().String("log.level", "debug", "log level")
	rootCmd.Flags().String("log.format", "console", "log format")
	rootCmd.Flags().String("log.output", "stdout", "log output")

	// 将每个 flag 绑定到 viper
	_ = viper.BindPFlag("log.level", rootCmd.Flags().Lookup("log.level"))
	_ = viper.BindPFlag("log.format", rootCmd.Flags().Lookup("log.format"))
	_ = viper.BindPFlag("log.output", rootCmd.Flags().Lookup("log.output"))
}

func initConfig() {
	// 设置配置文件
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("conf")
	}

	// 启用环境变量
	viper.SetEnvPrefix("CMD_RUNNER")
	viper.AutomaticEnv()

	// 将下划线/点号转换：如 MYAPP_SERVER_HOST → server.host
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	// 读取配置文件（可选，失败不退出）
	if err := viper.ReadInConfig(); err == nil {
		_, _ = fmt.Fprintf(os.Stderr, "Using config file: %s\n", viper.ConfigFileUsed())
	} else {
		// 如果没找到配置文件，也不报错（靠默认值或环境变量）
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			_, _ = fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		}
	}
}

func initLogger() {
	level := viper.GetString("log.level")
	format := viper.GetString("log.format")
	output := viper.GetString("log.output")

	var err error
	log, err = NewLogger(level, format, output)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
