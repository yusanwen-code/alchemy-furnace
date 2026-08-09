// seed 子命令: 写入内置金丹与默认供应商/模型种子(幂等)
package command

import (
	"fmt"

	"github.com/alchemy-furnace/server/internal/dao"
	"github.com/spf13/cobra"
)

// NewSeedCommand seed 子命令
func NewSeedCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "seed",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initDBForCommand(); err != nil {
				return err
			}
			defer dao.CloseDatabase()
			if err := dao.SeedBuiltinPills(dao.GetDB()); err != nil {
				return err
			}
			if err := dao.SeedDefaultLLMModels(dao.GetDB()); err != nil {
				return err
			}
			fmt.Println("[炼丹炉] 种子数据已就绪")
			return nil
		},
	}
}
