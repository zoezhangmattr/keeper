package services

type RunBackupResp struct {
	TaskId string `json:"taskId"`
}

type TaskProgressResp struct {
	Status      string `json:"status"`
	Description string `json:"description"`
	Message     string `json:"message"`
	Result      string `json:"result"`
	Progress    int32  `json:"progress"`
	ExportType  string `json:"exportType"`
}
