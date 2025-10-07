package g3cmd

// Deprecated: Use upload instead.
import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/calypr/data-client/data-client/commonUtils"
	"github.com/spf13/cobra"
)

func init() {
	var guid string
	var filePath string
	var bucketName string

	var uploadSingleCmd = &cobra.Command{
		Use:     "upload-single",
		Short:   "Upload a single file to a GUID",
		Long:    `Gets a presigned URL for which to upload a file associated with a GUID and then uploads the specified file.`,
		Example: `./data-client upload-single --profile=<profile-name> --guid=f6923cf3-xxxx-xxxx-xxxx-14ab3f84f9d6 --file=<path-to-file>`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Notice: this is the upload method which requires the user to provide a GUID. In this method file will be uploaded to a specified GUID.\nIf your intention is to upload file without pre-existing GUID, consider to use \"./data-client upload\" instead.\n\n")
			err := UploadSingle(profile, guid, filePath, bucketName)
			if err != nil {
				log.Fatalln(err.Error())
			}
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

func UploadSingle(profile string, guid string, filePath string, bucketName string) error {
	// disable logs
	log.SetOutput(io.Discard)

	// // initialize transmission logs
	// logs.InitSucceededLog(profile)
	// logs.InitFailedLog(profile)
	// logs.SetToBoth()
	// logs.InitScoreBoard(0)

	// Instantiate interface to Gen3
	gen3Interface := NewGen3Interface()

	// done so that profileConfig is written globally
	var err error
	profileConfig, err = conf.ParseConfig(profile)
	if err != nil {
		return err
	}

	filePaths, err := commonUtils.ParseFilePaths(filePath, false)
	if len(filePaths) > 1 {
		return errors.New("more than 1 file location has been found. Do not use \"*\" in file path or provide a folder as file path")
	}
	if err != nil {
		return errors.New("file path parsing error: " + err.Error())
	}
	if len(filePaths) == 1 {
		filePath = filePaths[0]
	}
	filename := filepath.Base(filePath)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// logs.AddToFailedLog(filePath, filename, commonUtils.FileMetadata{}, "", 0, false, true)
		// logs.IncrementScore(logs.ScoreBoardLen - 1)
		// logs.PrintScoreBoard()
		// logs.CloseAll()
		return fmt.Errorf("[ERROR] The file you specified \"%s\" does not exist locally", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		// logs.AddToFailedLog(filePath, filename, commonUtils.FileMetadata{}, "", 0, false, true)
		// logs.IncrementScore(logs.ScoreBoardLen - 1)
		// logs.PrintScoreBoard()
		// logs.CloseAll()
		// log.Fatalln("File open error: " + err.Error())
		return fmt.Errorf("[ERROR] when opening file path %s, an error occurred: %s", filePath, err.Error())
	}
	defer file.Close()

	furObject := commonUtils.FileUploadRequestObject{FilePath: filePath, Filename: filename, GUID: guid, Bucket: bucketName}

	furObject, err = GenerateUploadRequest(gen3Interface, furObject, file)
	if err != nil {
		file.Close()
		// logs.AddToFailedLog(furObject.FilePath, furObject.Filename, commonUtils.FileMetadata{}, furObject.GUID, 0, false, true)
		// logs.IncrementScore(logs.ScoreBoardLen - 1)
		// logs.PrintScoreBoard()
		// logs.CloseAll()
		// log.Fatalf("Error occurred during request generation: %s", err.Error())
		return fmt.Errorf("[ERROR] Error occurred during request generation for file %s: %s", filePath, err.Error())
	}
	err = uploadFile(furObject, 0)
	if err != nil {
		return fmt.Errorf("[ERROR] Error uploading file %s: %s", filePath, err.Error())
		// logs.IncrementScore(logs.ScoreBoardLen - 1) // update failed score
	} /*else {
	// logs.IncrementScore(0) // update succeeded score
	}*/
	// logs.PrintScoreBoard()
	// logs.CloseAll()
	return nil
}
