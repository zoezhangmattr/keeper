package main

import (
	"flag"
	"os"
	"strings"

	logger "github.com/sirupsen/logrus"
	"github.com/superwomany/keeper/services"
)

const (
	GET_LAST_TASK_ID = "last-task-id"
	RUN_BACKUP       = "run-backup"
	TASK_PROGRESS    = "task-progress"
	DOWNLOAD_LOCAL   = "download-local"
	BACKUP_SAVE      = "back-save"
)

func main() {
	mode := flag.String("mode", BACKUP_SAVE, "the name of the task")
	taskId := flag.String("id", "", "task id")
	fl := flag.String("link", "", "file link")
	flag.Parse()
	c := services.New()
	switch *mode {
	case BACKUP_SAVE:
		err := BackupSave(c)
		if err != nil {
			logger.Error(err)
			panic(err)
		}
	case GET_LAST_TASK_ID:
		id, err := c.GetLastTaskId()
		if err != nil {
			logger.Error(err)
			panic(err)
		}
		logger.Info(*id)
	case RUN_BACKUP:
		id, err := c.RunBackup()
		if err != nil {
			logger.Error(err)
			panic(err)
		}
		logger.Info(*id)
	case TASK_PROGRESS:
		if len(*taskId) == 0 {
			panic("task id is required")
		}
		c.EnsureTaskDone(*taskId)
	case DOWNLOAD_LOCAL:
		if len(*fl) == 0 {
			panic("file link is required")
		}
		fileId := strings.Split(*fl, "=")[1]
		logger.Info(fileId)
		out, err := os.Create(fileId + ".zip")
		if err != nil {
			logger.Error(err)
		}
		defer out.Close()
		c.Download(*fl, out)
	default:
		logger.Fatal("unknown mode")
	}

}

func BackupSave(c services.ServiceIntf) error {
	id, err := c.RunBackup()
	if err != nil {
		id, err = c.GetLastTaskId()
		if err != nil {
			return err
		}
	}
	dl := c.EnsureTaskDone(*id)
	if dl != nil {
		fileId := strings.Split(*dl, "=")[1]
		out, err := os.Create(fileId + ".zip")
		if err != nil {
			logger.Error(err)
			return err
		}
		defer out.Close()
		c.Download(*dl, out)
		if os.Getenv("BACKUP_BUCKET") != "" {
			input, err := os.Open(fileId + ".zip")
			if err != nil {
				logger.Error(err)
				return err
			}
			defer input.Close()
			s3k := "jira/" + fileId + ".zip"
			err = services.UploadS3(s3k, input)
			if err != nil {
				logger.Error(err)
				return err
			}
		}
	}
	return nil
}
