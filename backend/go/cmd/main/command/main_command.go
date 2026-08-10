// Package command cobra 子命令装配(对齐 Luna-CY 模板 cmd/main/command)
package command

import (
	"os"

	"github.com/alchemy-furnace/server/internal/configuration"
	"github.com/alchemy-furnace/server/internal/configuration/loader"
	"github.com/alchemy-furnace/server/internal/logger"
	"github.com/spf13/cobra"
)

// NewMainCommand 根命令;无参默认执行 serve
func NewMainCommand() *cobra.Command {
	command := &cobra.Command{
		Use:  "alchemy-server",
		Args: cobra.NoArgs,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if err := loader.LoadConfig(""); err != nil {
				cmd.PrintErrf("[炼丹炉] 加载配置失败: %v\n", err)
				os.Exit(1)
			}
			if err := logger.Init(configuration.Configuration.Server.Mode); err != nil {
				cmd.PrintErrf("[炼丹炉] 初始化日志失败: %v\n", err)
				os.Exit(1)
			}
		},
		// 无参默认 serve(兼容旧部署习惯)
		Run: func(cmd *cobra.Command, args []string) {
			runServe(cmd)
		},
	}

	command.AddCommand(NewServeCommand(), NewMigrateCommand(), NewSeedCommand())
	return command
}
