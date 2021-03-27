package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/damonchen/shugen/pkg/util/version"
	"github.com/dave/dst/decorator"
	"github.com/spf13/cobra"
)

var (
	inputFile   string
	outputFile  string
	showVersion bool
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&outputFile, "output", "o", "./shugen.go", "generate shu api file")
	rootCmd.PersistentFlags().BoolVarP(&showVersion, "version", "v", false, "version of shugen")

}

var (
	rootCmd = &cobra.Command{
		Use:   "shugen definition.go -o generator.go",
		Short: "shugen is the shu api generator (https://github.com/damonchen/shu)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				fmt.Println(version.Full())
				return nil
			}

			// 检测文件是否存在或者是否有权限访问
			_, err := os.Stat(inputFile)
			if err != nil {
				fmt.Println(err)
				return err
			}

			// Do not show command usage here.
			err = generatorAPI(inputFile, outputFile)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			return nil
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if showVersion {
				return nil
			}

			if len(args) == 0 {
				return errors.New("should given the define go file")
			}
			if len(args) > 1 {
				fmt.Println("[warning] given more than one file args, should only use first file")
			}
			inputFile = args[0]
			return nil
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func generatorAPI(inputFile string, outputFile string) (err error) {
	content, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return err
	}
	code := string(content)
	f, err := decorator.Parse(code)
	if err != nil {
		return err
	}

	pkg, err := extractClients(f)
	if err != nil {
		return err
	}
	fileContent := generatePkg(pkg)

	err = ioutil.WriteFile(outputFile, []byte(fileContent), 0644)
	if err != nil {
		return err
	}

	fmt.Printf("genereate %s done\n", outputFile)
	return nil
}
