# keeper
a small go app to backup jira

## usage locally
* add .env file to local, source .env
```.env
export JIRA_SITE=xxxx
export JIRA_USER=xxxx
export JIRA_PASSWORD=xxxx
export BACKUP_BUCKET=xxx
```
* run command
```sh
go run main.go -mode task-progress -id 10085
go run main.go -mode download-local -link "export/download/?fileId=82c61f81-e35e-4769-b1a7-3dba87b8e966"
go run main.go
```

## run in kubernetes
helm upgrade --install keeper --namespace keeper . -f values.yaml --set secret.jira_site=xxxxx \
 --set secret.jira_user=xxxx \
 --set secret.jira_password=xxxx
