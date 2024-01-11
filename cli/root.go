package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/umee-network/umeed-indexer/chain"
	"github.com/umee-network/umeed-indexer/database"
	"github.com/umee-network/umeed-indexer/idx"
	"github.com/umee-network/umeed-indexer/server"
)

// TODO: check to remove env and receive as flags with default value...
const (
	EnvChainRPC            = "CHAIN_RPC"
	EnvChainGRPC           = "CHAIN_GRPC"
	FlagMinimumBlockHeight = "block"
)

var (
	rootCmd = &cobra.Command{
		Use:   "umeed-indexer",
		Short: "A indexer for the umeed chain",
		Long:  `Basically index all the relevant tx, events and data from the umee blockchain`,
	}
)

// Execute executes the root command.
func Execute() error {
	printEnv()
	return rootCmd.Execute()
}

// init add the commands to the root.
func init() {
	rootCmd.AddCommand(CmdStartIndex())
	rootCmd.AddCommand(CmdDeleteChainData())
}

// CmdStartIndex start command line for start to listen to events and store chain data.
func CmdStartIndex() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Runs the indexer, querying and listening to the chain and storing it on the database.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			b, err := chain.NewBlockchain(os.Getenv(EnvChainRPC), os.Getenv(EnvChainGRPC))
			if err != nil {
				return err
			}

			logger, err := server.LoadLogger()
			if err != nil {
				fmt.Printf("Error loading logger: %s", err.Error())
				return err
			}

			ctx := context.Background()
			db, err := database.NewDB(database.Firebase, ctx, logger)
			if err != nil {
				return err
			}

			minimumBlockHeight, err := cmd.Flags().GetInt(FlagMinimumBlockHeight)
			if err != nil {
				return err
			}

			i, err := idx.NewIndexer(ctx, b, db, logger, minimumBlockHeight)
			if err != nil {
				return err
			}

			return i.Index(ctx)
		},
	}

	cmd.Flags().Int(FlagMinimumBlockHeight, 1, fmt.Sprintf("%s=100 to start indexing from block 100", FlagMinimumBlockHeight))
	return cmd
}

// CmdDeleteChainData only loads the database and deletes the chain data.
func CmdDeleteChainData() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [chain-id]",
		Short: "Connects to the database and deletes the chain data.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()

			logger, err := server.LoadLogger()
			if err != nil {
				fmt.Printf("Error loading logger: %s", err.Error())
				return err
			}

			db, err := database.NewDB(database.Firebase, ctx, logger)
			if err != nil {
				return err
			}
			defer db.Close()

			chainID := args[0]
			fmt.Printf("deleting chain data from db with chain-id: %s", chainID)
			return db.DeleteChainData(ctx, chainID)
		},
	}

	return cmd
}

// just prints the env file.
func printEnv() {
	rpc := os.Getenv(EnvChainRPC)
	grpc := os.Getenv(EnvChainGRPC)
	fmt.Printf(
		"__ENVS used__\n%s = %s\n%s = %s\n-----------------\n",
		EnvChainRPC, rpc,
		EnvChainGRPC, grpc,
	)
}
