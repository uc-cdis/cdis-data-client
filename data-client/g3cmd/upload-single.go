package g3cmd

// Deprecated: Use upload instead.
import (
	"errors"
	"fmt"
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
			UploadSingle(profile, guid, filePath, bucketName)
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
	// log something to log file called transfer.log
	f, err := os.OpenFile("transfer.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening log file: %w", err)
	}
	defer f.Close()
	log.SetOutput(f)
	log.Println("INSIDE: start upload single for file:", filePath, "with guid:", guid, "and bucket:", bucketName)

	// // initialize transmission logs
	// logs.InitSucceededLog(profile)
	// logs.InitFailedLog(profile)
	// logs.SetToBoth()
	// logs.InitScoreBoard(0)

	log.Println("INSIDE: logs initialized")

	// Instantiate interface to Gen3
	gen3Interface := NewGen3Interface()
	profileConfig, err = conf.ParseConfig(profile)
	if err != nil {
		return err
	}

	log.Println("INSIDE: profile config parsed")

	filePaths, err := commonUtils.ParseFilePaths(filePath, false)
	if len(filePaths) > 1 {
		errorStr := fmt.Sprintln("More than 1 file location has been found. Do not use \"*\" in file path or provide a folder as file path.")
		log.Println(errorStr)
		return errors.New(errorStr)
	}
	if err != nil {
		log.Println("File path parsing error: " + err.Error())
		return err
	}
	if len(filePaths) == 1 {
		filePath = filePaths[0]
	}
	filename := filepath.Base(filePath)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Println("ERROR:", filePath, filename, commonUtils.FileMetadata{}, "", 0, false, true)
		// logs.AddToFailedLog(filePath, filename, commonUtils.FileMetadata{}, "", 0, false, true)
		// logs.IncrementScore(logs.ScoreBoardLen - 1)
		// logs.PrintScoreBoard()
		// logs.CloseAll()
		errStr := fmt.Errorf("[ERROR] The file you specified \"%s\" does not exist locally.", filePath)
		log.Println(errStr.Error())
		return errStr
	}

	log.Println("INSIDE: file path parsed")

	file, err := os.Open(filePath)
	if err != nil {
		errorStr := fmt.Errorf("[ERROR] when opening file path %s, an error occurred: %s", filePath, err.Error())
		log.Println(errorStr.Error())
		// logs.AddToFailedLog(filePath, filename, commonUtils.FileMetadata{}, "", 0, false, true)
		// logs.IncrementScore(logs.ScoreBoardLen - 1)
		// logs.PrintScoreBoard()
		// logs.CloseAll()
		// log.Fatalln("File open error: " + err.Error())
		return errorStr
	}
	defer file.Close()

	log.Println("INSIDE: able to open file")

	furObject := commonUtils.FileUploadRequestObject{FilePath: filePath, Filename: filename, GUID: guid, Bucket: bucketName}

	furObject, err = GenerateUploadRequest(gen3Interface, furObject, file)
	if err != nil {
		file.Close()
		errorStr := fmt.Errorf("[ERROR] Error occurred during request generation for file %s: %s", filePath, err.Error())
		log.Println(errorStr.Error())
		// logs.AddToFailedLog(furObject.FilePath, furObject.Filename, commonUtils.FileMetadata{}, furObject.GUID, 0, false, true)
		// logs.IncrementScore(logs.ScoreBoardLen - 1)
		// logs.PrintScoreBoard()
		// logs.CloseAll()
		// log.Fatalf("Error occurred during request generation: %s", err.Error())
		return errorStr
	}
	err = uploadFile(furObject, 0)
	if err != nil {
		errStr := fmt.Errorf("[ERROR] Error uploading file %s: %s", filePath, err.Error())
		log.Println(errStr.Error())
		return errStr
		// logs.IncrementScore(logs.ScoreBoardLen - 1) // update failed score
	} /*else {
	// logs.IncrementScore(0) // update succeeded score
	}*/
	// logs.PrintScoreBoard()
	// logs.CloseAll()
	log.Println("INSIDE: upload complete")
	return nil
}
