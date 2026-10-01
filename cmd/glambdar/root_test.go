package glambdar

import (
	"bytes"
	"testing"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/spf13/cobra"
)

func TestRootCmd_Flags(t *testing.T) {
	cmd := RootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Test persistent flag existence
	flags := []string{
		"db-type",
		"dsn",
		"s3-endpoint",
		"s3-region",
		"s3-bucket",
		"s3-access-key-id",
		"s3-secret-access-key",
		"s3-session-token",
		"s3-force-path-style",
	}

	for _, f := range flags {
		if cmd.PersistentFlags().Lookup(f) == nil {
			t.Errorf("expected persistent flag %q to be registered", f)
		}
	}
}

func TestAgentCmd_Flags(t *testing.T) {
	// Test local flag existence on agent subcommand
	agentFlags := []string{
		"grpc-port",
		"node-id",
		"redis-addr",
		"worker-path",
	}

	for _, f := range agentFlags {
		if agentCmd.Flags().Lookup(f) == nil {
			t.Errorf("expected agent flag %q to be registered", f)
		}
	}
}

func TestApplyPersistentFlags(t *testing.T) {
	testCmd := &cobra.Command{Use: "test"}
	testCmd.Flags().StringVar(&flagDBType, "db-type", "", "")
	testCmd.Flags().StringVar(&flagDSN, "dsn", "", "")
	_ = testCmd.Flags().Parse([]string{"--db-type", "mysql", "--dsn", "user:pass@/mydb"})

	applyPersistentFlags(testCmd)

	if config.Overrides.DBType != "mysql" {
		t.Errorf("expected config.Overrides.DBType to be 'mysql', got %q", config.Overrides.DBType)
	}
	if config.Overrides.DSN != "user:pass@/mydb" {
		t.Errorf("expected config.Overrides.DSN to be 'user:pass@/mydb', got %q", config.Overrides.DSN)
	}
}
