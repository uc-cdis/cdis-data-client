package g3cmd

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

func computeMD5(file string) string {
	h := md5.New()
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if _, err := io.Copy(h, f); err != nil {
		log.Fatal(err)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func init() {
	var template string
	var output string

	var generateTSVCmd = &cobra.Command{
		Use:     "generate-tsv",
		Short:   "Generate a file upload tsv from a template",
		Long:    `Fills in a Gen3 data file template with information from a directory of files.`,
		Example: `./gen3-client generate-tsv --from-template=image_file.tsv files*.dcm`,
		Args:    cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// don't initialize transmission logs for non-uploading related commands
			logs.SetToBoth()

			outFile, err := os.Create(output)
			if err != nil {
				log.Fatalf(err.Error())
			}
			defer outFile.Close()

			csvfile, err := os.Open(template)
			if err != nil {
				log.Fatalf("Failed to read template csv file %v", template)
			}
			defer csvfile.Close()

			csvReader := csv.NewReader(csvfile)
			csvReader.Comma = '\t'
			headers, err := csvReader.Read()
			if err != nil {
				log.Fatalf("Failed to read template csv file %v", template)
			}

			for index, header := range headers {
				addTab := ""
				if index != 0 {
					addTab = "\t"
				}
				fmt.Fprintf(outFile, "%s%s", addTab, header)
			}
			fmt.Fprintf(outFile, "\n")
			files, err := filepath.Glob(args[0])
			if err != nil {
				log.Fatalf(err.Error())
			}

			for _, file := range files {
				fmt.Printf("Adding file %v\n", file)
				for index, header := range headers {
					addTab := ""
					if index != 0 {
						addTab = "\t"
					}
					outputString := ""
					if header == "*file_name" {
						outputString = filepath.Base(file)
					} else if header == "*md5sum" {
						outputString = computeMD5(file)
					} else if header == "*file_size" {
						fileInfo, err := os.Stat(file)
						if err != nil {
							log.Fatalf(err.Error())
						}
						outputString = fmt.Sprintf("%v", fileInfo.Size())
					}
					fmt.Fprintf(outFile, "%s%s", addTab, outputString)
				}
				fmt.Fprintf(outFile, "\n")
			}

			fmt.Printf("Generated tsv %v from files %v!\n", output, args[0])
			logs.CloseMessageLog()
		},
	}

	generateTSVCmd.Flags().StringVar(&template, "from-template", "", "The template tsv to read from")
	generateTSVCmd.MarkFlagRequired("from-template")
	generateTSVCmd.Flags().StringVar(&output, "output", "", "The output tsv to write")
	generateTSVCmd.MarkFlagRequired("output")
	RootCmd.AddCommand(generateTSVCmd)
}
