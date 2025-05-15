package flipcam

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
)

var versionRequested = false

var rootCmd = &cobra.Command{
	Use:               "flipcam",
	DisableAutoGenTag: true,
	Run: func(cmd *cobra.Command, args []string) {
		if versionRequested {
			fmt.Println("flipcam version 0.1.0")
			return
		}

		err := cmd.Help()
		if err != nil {
			log.Fatalf("Error printing help information: %v\n", err)
		}
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func GetCommand() *cobra.Command {
	return rootCmd
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.Flags().BoolVar(&versionRequested, "version", false, "Version info")
}
