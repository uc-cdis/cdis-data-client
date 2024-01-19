package g3cmd

// Deprecated: Use upload instead.
import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/uc-cdis/gen3-client/gen3-client/logs"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

func init() {
	var guid string
	var filePath string
	var bucketName string

	var uploadSingleCmd = &cobra.Command{
		Use:     "upload-single",
		Short:   "Upload a single file to a GUID",
		Long:    `Gets a presigned URL for which to upload a file associated with a GUID and then uploads the specified file.`,
		Example: `./gen3-client upload-single --profile=<profile-name> --guid=f6923cf3-xxxx-xxxx-xxxx-14ab3f84f9d6 --file=<path-to-file>`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Notice: this is the upload method which requires the user to provide a GUID. In this method file will be uploaded to a specified GUID.\nIf your intention is to upload file without pre-existing GUID, consider to use \"./gen3-client upload\" instead.\n\n")

			// initialize transmission logs
			logs.InitSucceededLog(profile)
			logs.InitFailedLog(profile)
			logs.SetToBoth()
			logs.InitScoreBoard(0)

			// Instantiate interface to Gen3
			gen3Interface := NewGen3Interface()
			profileConfig = conf.ParseConfig(profile)

			filePaths, err := commonUtils.ParseFilePaths(filePath, false)
			if len(filePaths) > 1 {
				fmt.Println("More than 1 file location has been found. Do not use \"*\" in file path or provide a folder as file path.")
				return
			}
			if err != nil {
				log.Fatalln("File path parsing error: " + err.Error())
			}
			if len(filePaths) == 1 {
				filePath = filePaths[0]
			}
			filename := filepath.Base(filePath)
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				logs.AddToFailedLog(filePath, filename, commonUtils.FileMetadata{}, "", 0, false, true)
				logs.IncrementScore(logs.ScoreBoardLen - 1)
				logs.PrintScoreBoard()
				logs.CloseAll()
				log.Fatalf("The file you specified \"%s\" does not exist locally.", filePath)
			}

			file, err := os.Open(filePath)
			if err != nil {
				logs.AddToFailedLog(filePath, filename, commonUtils.FileMetadata{}, "", 0, false, true)
				logs.IncrementScore(logs.ScoreBoardLen - 1)
				logs.PrintScoreBoard()
				logs.CloseAll()
				log.Fatalln("File open error: " + err.Error())
			}
			defer file.Close()

			furObject := commonUtils.FileUploadRequestObject{FilePath: filePath, Filename: filename, GUID: guid, Bucket: bucketName}

			furObject, err = GenerateUploadRequest(gen3Interface, furObject, file)
			if err != nil {
				file.Close()
				logs.AddToFailedLog(furObject.FilePath, furObject.Filename, commonUtils.FileMetadata{}, furObject.GUID, 0, false, true)
				logs.IncrementScore(logs.ScoreBoardLen - 1)
				logs.PrintScoreBoard()
				logs.CloseAll()
				log.Fatalf("Error occurred during request generation: %s", err.Error())
			}
			err = uploadFile(furObject, 0)
			if err != nil {
				log.Println(err.Error())
				logs.IncrementScore(logs.ScoreBoardLen - 1) // update failed score
			} else {
				logs.IncrementScore(0) // update succeeded score
			}
			logs.PrintScoreBoard()
			logs.CloseAll()
		},
	}

	uploadSingleCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	uploadSingleCmd.MarkFlagRequired("profile") //nolint:errcheck
	uploadSingleCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	uploadSingleCmd.MarkFlagRequired("guid") //nolint:errcheck
	uploadSingleCmd.Flags().StringVar(&filePath, "file", "", "Specify file to upload to with --file=~/path/to/file")
	uploadSingleCmd.MarkFlagRequired("file") //nolint:errcheck
	uploadSingleCmd.Flags().StringVar(&bucketName, "bucket", "", "The bucket to which files will be uploaded. If not provided, defaults to Gen3's configured DATA_UPLOAD_BUCKET.")
	RootCmd.AddCommand(uploadSingleCmd)
}
