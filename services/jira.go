package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	logger "github.com/sirupsen/logrus"
)

const (
	RUN_BACKUP_URI    = "/rest/backup/1/export/runbackup"
	LAST_TASK_ID_URI  = "/rest/backup/1/export/lastTaskId"
	TASK_PROGRESS_URI = "/rest/backup/1/export/getProgress"
	SERVLET_URI       = "/plugins/servlet"
)

// HTTPDoer describes something that performs an http request.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}
type Config struct {
	JiraSite     string
	JiraUser     string
	JiraPassword string
}
type client struct {
	config *Config
	doer   HTTPDoer
}

type ServiceIntf interface {
	RunBackup() (*string, error)
	GetLastTaskId() (*string, error)
	GetProgressOfTask(string) (*int32, *string, error)
	EnsureTaskDone(string) *string
	Download(string, io.Writer) error
}

func New() ServiceIntf {
	c := &Config{
		JiraSite:     os.Getenv("JIRA_SITE"),
		JiraUser:     os.Getenv("JIRA_USER"),
		JiraPassword: os.Getenv("JIRA_PASSWORD"),
	}
	return &client{
		config: c,
		doer:   &http.Client{},
	}
}

func NewWith(c *Config, h HTTPDoer) ServiceIntf {
	return &client{
		config: c,
		doer:   h,
	}
}

// RunBackup triggers backup and returns a taskId
func (c *client) RunBackup() (*string, error) {
	uri := "https://" + c.config.JiraSite + RUN_BACKUP_URI
	payload := map[string]string{
		"cbAttachments": "true",
		"exportToCloud": "true",
	}
	buf := &bytes.Buffer{}
	err := json.NewEncoder(buf).Encode(payload)
	if err != nil {
		logger.WithFields(logger.Fields{
			"url":     uri,
			"error":   err,
			"payload": payload,
		}).Errorf("json encode failed.")
		return nil, err
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, uri, buf)

	result := RunBackupResp{}
	if err != nil {
		logger.WithFields(logger.Fields{
			"url":   uri,
			"error": err,
		}).Errorf("request failed.")
		return nil, err
	}
	err = c.DoRequest(c.doer, req, &result)
	if err != nil {
		logger.WithFields(logger.Fields{
			"url":   uri,
			"error": err,
		}).Errorf("request failed.")
		return nil, err
	}
	logger.WithFields(logger.Fields{
		"url":    uri,
		"taskId": result.TaskId,
	}).Info("request succeed")
	return &result.TaskId, nil
}

// GetLastTaskId retrieves the last task id of the runbackup
func (c *client) GetLastTaskId() (*string, error) {

	uri := "https://" + c.config.JiraSite + LAST_TASK_ID_URI

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, uri, nil)
	if err != nil {
		logger.WithFields(logger.Fields{
			"url":   uri,
			"error": err,
		}).Errorf("request failed.")
		return nil, err
	}

	var result int32
	err = c.DoRequest(c.doer, req, &result)
	if err != nil {
		logger.WithFields(logger.Fields{
			"url":   uri,
			"error": err,
		}).Errorf("request failed.")
		return nil, err
	}
	logger.WithFields(logger.Fields{
		"url":    uri,
		"taskId": result,
	}).Info("request succeed.")
	str := strconv.FormatInt(int64(result), 10)

	return &str, nil
}

// GetLastTaskId retrieves the progress of task and the result
func (c *client) GetProgressOfTask(id string) (*int32, *string, error) {

	uri := "https://" + c.config.JiraSite + TASK_PROGRESS_URI

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, uri, nil)
	q := req.URL.Query()
	q.Add("taskId", id)
	req.URL.RawQuery = q.Encode()
	if err != nil {
		logger.WithFields(logger.Fields{
			"url":    uri,
			"error":  err,
			"taskId": id,
		}).Errorf("request failed.")
		return nil, nil, err
	}

	result := TaskProgressResp{}
	err = c.DoRequest(c.doer, req, &result)
	if err != nil {
		logger.WithFields(logger.Fields{
			"url":    uri,
			"error":  err,
			"taskId": id,
		}).Errorf("request failed.")
		return nil, nil, err
	}
	logger.WithFields(logger.Fields{
		"url":      uri,
		"status":   result.Status,
		"progress": result.Progress,
		"taskId":   id,
	}).Info("request succeed.")

	return &result.Progress, &result.Result, nil
}

// EnsureTaskDone checks task progress until it is 100 and return the result
func (c *client) EnsureTaskDone(id string) *string {

	logger.Info("check task progress")
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	progressChan := make(chan TaskProgressResp)
	errorChan := make(chan error)

	for {
		select {
		case <-ticker.C:
			go func(pc chan TaskProgressResp, ec chan error) {
				progress, result, err := c.GetProgressOfTask(id)
				if err != nil {
					errorChan <- err
					return
				}
				pc <- TaskProgressResp{Progress: *progress, Result: *result}
			}(progressChan, errorChan)
		case p := <-progressChan:
			logger.Infof("progress=%v", p.Progress)
			if p.Progress == 100 {
				logger.Infof("result=%s", p.Result)
				return &p.Result
			}
		case <-errorChan:
			return nil
		}
	}

}

func (c *client) Download(fl string, out io.Writer) error {
	uri := "https://" + c.config.JiraSite + SERVLET_URI + "/" + fl

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, uri, nil)

	if err != nil {
		logger.WithFields(logger.Fields{
			"url":   uri,
			"error": err,
		}).Errorf("request failed")
		return err
	}

	err = c.DoRequest(c.doer, req, out)
	if err != nil {
		logger.WithFields(logger.Fields{
			"url":   uri,
			"error": err,
		}).Errorf("request failed")
		return err
	}
	logger.WithFields(logger.Fields{
		"url": uri,
	}).Info("request succeed.")

	return nil
}

// basicAuth generates base64 encode auth token
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// DoRequest sends http request
func (c *client) DoRequest(doer HTTPDoer, req *http.Request, dst interface{}) error {
	req.Header.Set("Authorization", "Basic "+basicAuth(c.config.JiraUser, c.config.JiraPassword))
	req.Header.Set("Content-Type", "application/json")
	resp, err := doer.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-200 OK status code: %v response body: %q", resp.Status, resp.Body)
	}
	if w, ok := dst.(io.Writer); ok {
		_, err := io.Copy(w, resp.Body)
		if err != nil {
			return err
		}
	} else {
		raw, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if err := json.Unmarshal(raw, dst); err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}
