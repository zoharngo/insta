# Apex

Apex performs the following upon `apex init`:

* creating IAM PhotosApp_lambda_function role
* creating IAM PhotosApp_lambda_logs policy
* attaching policy to lambda_function r

## Build

`apex -l debug build thumbnail > thumbnail.zip`

## Deploy

`apex deploy`

## Create Lambda Trigger

1. Lambda > Functions > PhotosApp_thumbnail > Triggers > Add trigger
1. Click empty box
1. Select S3
1. Choose bucket (i.e. "pluralsight-photos")
1. Choose event type "Object created (All)"
1. Click Submit

## Edit IAM policy to allow S3 full access

1. IAM > Roles > PhotosApp_lambda_function
1. Permissions > Attach policy
1. Select "AmazonS3FullAccess" checkbox
1. Click "Attach policy"

## Test Lambda Function

1. Upload an image to the bucket in S3

## View CloudWatch Logs

