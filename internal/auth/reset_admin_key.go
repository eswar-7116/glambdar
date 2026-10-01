package auth

import (
	"fmt"
	"os"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	"github.com/spf13/cobra"
)

// Reset the root admin key
var ResetAdminKeyCmd = &cobra.Command{
	Use:   "reset-admin-key",
	Short: "Reset root admin API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.InitPaths(); err != nil {
			fmt.Println(err.Error())
			fmt.Println("Failed to initialize paths.")
			os.Exit(1)
		}

		if err := config.DB.AutoMigrate(&functions.Metadata{}, &functions.Log{}, &APIKey{}, &AuditLog{}); err != nil {
			fmt.Println("Failed to migrate database schema:", err)
			os.Exit(1)
		}

		if err := ResetRootKey(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to reset admin key: %v\n", err)
			return err
		}
		return nil
	},
}
