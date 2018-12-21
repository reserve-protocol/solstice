package cmd

import (
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var cfgFile string
var txnHash string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
    Use:   "solstice",
    Short: "Code coverage and debugging tools for Solidity",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

func init() {
    cobra.OnInitialize(initConfig)

    // Here you will define your flags and configuration settings.
    // Cobra supports persistent flags, which, if defined here,
    // will be global for your application.
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
    workingDir, err := os.Getwd()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }

    viper.AddConfigPath(workingDir)

    if cfgFile != "" {
        // Use config file from the flag.
        viper.SetConfigFile(cfgFile)
    }

    viper.AutomaticEnv() // read in environment variables that match

    // If a config file is found, read it in.
    err = viper.ReadInConfig()
    switch err.(type) {
    case viper.ConfigFileNotFoundError:
        // Ok if no config file.
    default:
        check(err, "failed to read config file")
    }
}

func check(err error, msg string) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "%v: %v\n", msg, err)
        os.Exit(1)
    }
}
