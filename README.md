# yc-url-shrinker

yc-url-shrinker is an application which makes short URL and stores results into YDB.

You can use Yandex Cloud to deploy it.

## Usage
You can use it as a HTTP server, or as a serverless function.

### Running as a http server
First of all you need to set the path to the key of your service account in environment.
For example:
```
export YDB_SERVICE_ACCOUNT_KEY_FILE_CREDENTIALS=/path/to/sa/key/file
```
Then you can build and run the application:
```
git clone https://github.com/prgr4xmpl/yc-url-shrinker.git
go build -o yc-url-shrinker .
yc-url-shrinker -ydb=grpcs://your.path.to.ydb.net
```

### Running as a serverless function
First you must create a go.mod file, zip files and upload the function
```
go mod init yc-url-shrinker && go mod tidy
zip yc-url-shrinker service.go go.mod go.sum
yc serverless function version create \
   --runtime=golang121 \
   --entrypoint=service.Serverless \
   --memory=128m \
   --execution-timeout=1s \
   --environment YDB="<YDB_ENDPOINT>" \
   --source-path=./yc-url-shrinker.zip \
   --function-name=yc-url-shrinker \
   --service-account-id=<SERVICE_ACCOUNT_ID>
```
Then you must create a API Gateway.

You can use the example file in openapi folder of repo.

Now your service is ready to go.
