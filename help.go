package main

import "fmt"

func printHelp() {
    fmt.Println(`Usage: glambdar [flags]

Configuration can be provided via config.json, environment variables (GLMBD_*) or CLI flags. CLI flags have highest precedence.

Available flags:
  --db_type string               Database type (postgres, mysql)
  --dsn string                   Data source name
  --s3_endpoint string           S3 endpoint URL
  --s3_region string             S3 region
  --s3_bucket string             S3 bucket name
  --s3_access_key_id string      S3 access key ID
  --s3_secret_access_key string  S3 secret access key
  --s3_session_token string      S3 session token
  --s3_force_path_style bool     Force path style for S3
  --help, -h                     Show this help message
  --version, -v                  Show version

Run 'glambdar --version' for version.`)
}
