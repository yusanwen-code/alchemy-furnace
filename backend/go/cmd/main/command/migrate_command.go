// migrate 子命令: migrate up(执行全部未应用迁移) / migrate down(回滚全部,带确认)
package command

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alchemy-furnace/server/internal/configuration"
	"github.com/alchemy-furnace/server/internal/dao"
	"github.com/spf13/cobra"
)

// NewMigrateCommand migrate 命令组
func NewMigrateCommand() *cobra.Command {
	command := &cobra.Command{
		Use:  "migrate",
		Args: cobra.NoArgs,
	}
	command.AddCommand(newMigrateUpCommand(), newMigrateDownCommand())
	return command
}

// initDBForCommand 子命令通用: 初始化数据库连接
func initDBForCommand() error {
	return dao.InitDatabase(&configuration.Configuration.Database)
}

func newMigrateUpCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "up",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDBForCommand(); err != nil {
				return err
			}
			defer dao.CloseDatabase()
			if err := dao.MigrateUp(); err != nil {
				return err
			}
			v, dirty, _ := dao.MigrateVersion()
			fmt.Printf("[炼丹炉] 迁移完成,当前版本: %d (dirty=%v)\n", v, dirty)
			return nil
		},
	}
}

func newMigrateDownCommand() *cobra.Command {
	var yes bool
	command := &cobra.Command{
		Use:  "down",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Print("[炼丹炉] 警告: 将 DROP 全部业务表,数据不可恢复!输入 yes 确认: ")
				line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				if strings.TrimSpace(line) != "yes" {
					fmt.Println("[炼丹炉] 已取消")
					return nil
				}
			}
			if err := initDBForCommand(); err != nil {
				return err
			}
			defer dao.CloseDatabase()
			if err := dao.MigrateDown(); err != nil {
				return err
			}
			fmt.Println("[炼丹炉] 已回滚全部迁移")
			return nil
		},
	}
	command.Flags().BoolVarP(&yes, "yes", "y", false, "跳过确认(脚本用)")
	return command
}
