// Package controller defines Cobra command factories for the CLI client.
//
// common.go holds output-format state, cross-resource helpers, and all
// command factory functions (task and admin).
package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/ryo-arima/cmn-weave/pkg/client/usecase"
	"github.com/spf13/cobra"
)

// outputFormat is the process-wide output format (table / json / yaml).
var outputFormat = "table"

// SetOutputFormat sets the global output format.
// Unrecognised values fall back to "table".
func SetOutputFormat(format string) {
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "table", "json", "yaml":
		outputFormat = format
	default:
		outputFormat = "table"
	}
}

// GetOutputFormat returns the current output format.
func GetOutputFormat() string { return outputFormat }

// ---------------------------------------------------------------------------
// Admin commands
// ---------------------------------------------------------------------------

// NewBootstrapDBCmd returns the "bootstrap db" sub-command.
// It connects directly to PostgreSQL using the DSN from client config and
// creates all required tables if they do not already exist.
func NewBootstrapDBCmd(uc *usecase.CommonUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "db",
		Short: "Bootstrap the database schema (creates tables if they do not exist)",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := uc.BootstrapDB(context.Background())
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), usecase.Format(GetOutputFormat(), resp))
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// Task commands
// ---------------------------------------------------------------------------

// NewDispatchCmd returns the "dispatch" subcommand.
func NewDispatchCmd(uc *usecase.CommonUsecase) *cobra.Command {
	var argv []string
	var env []string
	cmd := &cobra.Command{
		Use:   "dispatch",
		Short: "Dispatch a new task to an agent via the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := uc.Dispatch(context.Background(), argv, env)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), usecase.Format(GetOutputFormat(), map[string]string{
				"task_id": taskID,
			}))
			return nil
		},
	}
	cmd.Flags().StringSliceVar(&argv, "argv", nil, "command and arguments to execute on the agent")
	cmd.Flags().StringSliceVar(&env, "env", nil, "environment variables (KEY=VALUE) for the task")
	return cmd
}

// NewStatusCmd returns the "status" subcommand.
func NewStatusCmd(uc *usecase.CommonUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "status <task_id>",
		Short: "Query the status of a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := uc.Status(context.Background(), args[0])
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), usecase.Format(GetOutputFormat(), resp))
			return nil
		},
	}
}

// NewCancelCmd returns the "cancel" subcommand.
func NewCancelCmd(uc *usecase.CommonUsecase) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <task_id>",
		Short: "Request cancellation of a running task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := uc.Cancel(context.Background(), args[0]); err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), usecase.Format(GetOutputFormat(), map[string]string{
				"task_id": args[0],
				"status":  "cancel requested",
			}))
			return nil
		},
	}
}

