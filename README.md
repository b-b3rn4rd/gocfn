[![Go Report Card](https://goreportcard.com/badge/github.com/b-b3rn4rd/gocfn)](https://goreportcard.com/report/github.com/b-b3rn4rd/gocfn)  [![Build Status](https://travis-ci.org/b-b3rn4rd/gocfn.svg?branch=master)](https://travis-ci.org/b-b3rn4rd/gocfn) *GOCFN* - cloudformation package and deploy commands in Golang
==================
Following library re-implements existing functionality in `aws cloudformation package` and `aws cloudformation deploy` and provides additional features that are not available in the standard `aws cli`

Motivation
----------
I was really frustrated with the behaviour of the standard `aws cloudformation deploy` command, particularly with following issues 

(*at the time I started writing this library*):

1) No options to specify `--tags`
2) Command would fail if change set does not contain changes
3) Command does not output describe stack on successful completion

In addition, to fixing above mentioned issues I also needed those features:
1) Preview option - to generate and output a change set without actually executing it
2) Stream option - stream stack events while stack is being executed
3) Force deploy option - delete stack if it's failed creation and in `CREATE_FAILED` or `ROLLBACK_COMPLETE` status

Installation
--------------
There are three ways to install `gocfn`

*Using homebrew*

`brew install gocfn`

*Using go get*

`go get github.com/b-b3rn4rd/gocfn`


*Manually*

Download and install a binary the releases page.


Usage
------------------
*gocfn deploy* - provides identical parameters to `aws cloudformation deploy` and can be transparently substitute it.


```bash
usage: gocfn deploy --template-file=TEMPLATE-FILE --name=NAME [<flags>]

Deploys the specified AWS CloudFormation template by creating and then executing a change set.

Flags:
      --help                     Show context-sensitive help (also try --help-long and --help-man).
  -d, --debug                    Enable debug logging.
      --version                  Show application version.
      --template-file=TEMPLATE-FILE  
                                 The path where your AWS CloudFormation template is located.
      --name=NAME                The name of the AWS CloudFormation stack you're deploying to.
      --s3-bucket=S3-BUCKET      The name of the S3 bucket where this command uploads your CloudFormation template.
      --force-upload             Indicates whether to override existing files in the S3 bucket.
      --s3-prefix=S3-PREFIX      A prefix name that the command adds to the artifacts name when it uploads them to the S3 bucket.
      --kms-key-id=KMS-KEY-ID    The ID of an AWS KMS key that the command uses to encrypt artifacts that are at rest in the S3 bucket.
      --parameter-overrides=PARAMETER-OVERRIDES  
                                 A list of parameter structures that specify input parameters for your stack template.
      --capabilities=CAPABILITIES ...  
                                 A list of capabilities that you must specify before AWS Cloudformation can create certain stacks.
      --no-execute-changeset     Indicates whether to execute the change set. Specify this flag if you want to view your stack changes before executing
      --role-arn=ROLE-ARN        The Amazon Resource Name (ARN) of an AWS Identity and Access Management (IAM) role
      --notification-arns=NOTIFICATION-ARNS ...  
                                 The Amazon Resource Name (ARN) of an AWS Identity and Access Management (IAM) role.
      --fail-on-empty-changeset  Specify if the CLI should return a non-zero exit code if there are no changes to be made to the stack
      --tags=TAGS                A list of tags to associate with the stack that is created or updated.
      --force-deploy             Force CloudFormation stack deployment if it's in CREATE_FAILED state.
      --stream                   Stream stack events during creation or update process.
```

Examples
------------

<details>
<summary>Generate a change set without actually executing it</summary>


Change set can be generated without execution by passing `--no-execute-changeset` parameter, the output will be written
to the `stdout`

```bash
$ gocfn deploy --name hello --parameter-overrides "BucketName=helloza" --template-file stack.yml --no-execute-changeset
{
    "Capabilities": null,
    "ChangeSetId": "arn:aws:cloudformation:us-west-2:111111111111:changeSet/gocfn-cloudformation-package-deploy-1521960705/f229d566-fc74-456d-8ad4-5cb7458c8411",
    "ChangeSetName": "gocfn-cloudformation-package-deploy-1521960705",
    "Changes": [
        {
            "ResourceChange": {
                "Action": "Modify",
                "Details": [
                    {
                        "CausingEntity": "S3Bucket",
                        "ChangeSource": "ResourceReference",
                        "Evaluation": "Static",
                        "Target": {
                            "Attribute": "Properties",
                            "Name": "Bucket",
                            "RequiresRecreation": "Always"
                        }
                    },
                    {
                        "CausingEntity": "S3Bucket",
                        "ChangeSource": "ResourceReference",
                        "Evaluation": "Static",
                        "Target": {
                            "Attribute": "Properties",
                            "Name": "PolicyDocument",
                            "RequiresRecreation": "Never"
                        }
                    }
                ],
                "LogicalResourceId": "BucketPolicy",
                "PhysicalResourceId": "hello-BucketPolicy-15UXUJSQ48KAH",
                "Replacement": "True",
                "ResourceType": "AWS::S3::BucketPolicy",
                "Scope": [
                    "Properties"
                ]
            },
            "Type": "Resource"
        },
        {
            "ResourceChange": {
                "Action": "Modify",
                "Details": [
                    {
                        "CausingEntity": null,
                        "ChangeSource": "DirectModification",
                        "Evaluation": "Dynamic",
                        "Target": {
                            "Attribute": "Properties",
                            "Name": "BucketName",
                            "RequiresRecreation": "Always"
                        }
                    },
                    {
                        "CausingEntity": "BucketName",
                        "ChangeSource": "ParameterReference",
                        "Evaluation": "Static",
                        "Target": {
                            "Attribute": "Properties",
                            "Name": "BucketName",
                            "RequiresRecreation": "Always"
                        }
                    }
                ],
                "LogicalResourceId": "S3Bucket",
                "PhysicalResourceId": "gellozaa",
                "Replacement": "True",
                "ResourceType": "AWS::S3::Bucket",
                "Scope": [
                    "Properties"
                ]
            },
            "Type": "Resource"
        }
    ],
    "CreationTime": "2018-03-25T06:51:47.628Z",
    "Description": "Created by gocfn at 2018-03-25 06:51:45.301595904 +0000 UTC",
    "ExecutionStatus": "AVAILABLE",
    "NextToken": null,
    "NotificationARNs": null,
    "Parameters": [
        {
            "ParameterKey": "BucketName",
            "ParameterValue": "helloza",
            "ResolvedValue": null,
            "UsePreviousValue": null
        }
    ],
    "RollbackConfiguration": {
        "MonitoringTimeInMinutes": null,
        "RollbackTriggers": null
    },
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Status": "CREATE_COMPLETE",
    "StatusReason": null,
    "Tags": null
}

```
</details>



<details>
<summary>(Re)deploy a stack that does not contain changes</summary>

In situations when stack does not contain new changes, `gocfn` won't fail and output stack's information, unless
`--fail-on-empty-changeset` is specified.

```bash
gocfn deploy --name hello --parameter-overrides "BucketName=helloza" --template-file stack.yml
{
    "Capabilities": null,
    "ChangeSetId": "arn:aws:cloudformation:us-west-2:111111111111:changeSet/gocfn-cloudformation-package-deploy-1521960952/0db34469-ba57-4286-b5c7-ff049763c5fb",
    "CreationTime": "2018-03-01T08:57:19.888Z",
    "DeletionTime": null,
    "Description": null,
    "DisableRollback": false,
    "EnableTerminationProtection": false,
    "LastUpdatedTime": "2018-03-25T06:56:25.743Z",
    "NotificationARNs": null,
    "Outputs": [
        {
            "Description": "Name of S3 bucket to hold website content",
            "ExportName": null,
            "OutputKey": "S3BucketSecureURL",
            "OutputValue": "https://helloza.s3.amazonaws.com"
        },
        {
            "Description": "URL for website hosted on S3",
            "ExportName": null,
            "OutputKey": "WebsiteURL",
            "OutputValue": "http://helloza.s3-website-us-west-2.amazonaws.com"
        }
    ],
    "Parameters": [
        {
            "ParameterKey": "BucketName",
            "ParameterValue": "helloza",
            "ResolvedValue": null,
            "UsePreviousValue": null
        }
    ],
    "ParentId": null,
    "RoleARN": null,
    "RollbackConfiguration": {
        "MonitoringTimeInMinutes": null,
        "RollbackTriggers": null
    },
    "RootId": null,
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "StackStatus": "UPDATE_COMPLETE",
    "StackStatusReason": null,
    "Tags": null,
    "TimeoutInMinutes": null
}
```

Error on empty change set can be forced by specifying `--fail-on-empty-changeset`

```bash
gocfn deploy --name hello --parameter-overrides "BucketName=helloza" --template-file stack.yml --fail-on-empty-changeset
{"error":"The submitted information didn't contain changes. Submit different information to create a change set.","level":"error","msg":"ChangeSet creation error","time":"2018-03-25T18:00:16+11:00"}
```
</details>



<details>
<summary>Deploy stack from s3 bucket</summary>

```bash
gocfn deploy --name hello --parameter-overrides "BucketName=helloza" --template-file stack.yml --s3-bucket cf-templates-17636j9pul1d8-us-west-2
{
    "Capabilities": null,
    "ChangeSetId": "arn:aws:cloudformation:us-west-2:111111111111:changeSet/gocfn-cloudformation-package-deploy-1521960952/0db34469-ba57-4286-b5c7-ff049763c5fb",
    "CreationTime": "2018-03-01T08:57:19.888Z",
    "DeletionTime": null,
    "Description": null,
    "DisableRollback": false,
    "EnableTerminationProtection": false,
    "LastUpdatedTime": "2018-03-25T06:56:25.743Z",
    "NotificationARNs": null,
    "Outputs": [
        {
            "Description": "Name of S3 bucket to hold website content",
            "ExportName": null,
            "OutputKey": "S3BucketSecureURL",
            "OutputValue": "https://helloza.s3.amazonaws.com"
        },
        {
            "Description": "URL for website hosted on S3",
            "ExportName": null,
            "OutputKey": "WebsiteURL",
            "OutputValue": "http://helloza.s3-website-us-west-2.amazonaws.com"
        }
    ],
    "Parameters": [
        {
            "ParameterKey": "BucketName",
            "ParameterValue": "helloza",
            "ResolvedValue": null,
            "UsePreviousValue": null
        }
    ],
    "ParentId": null,
    "RoleARN": null,
    "RollbackConfiguration": {
        "MonitoringTimeInMinutes": null,
        "RollbackTriggers": null
    },
    "RootId": null,
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "StackStatus": "UPDATE_COMPLETE",
    "StackStatusReason": null,
    "Tags": null,
    "TimeoutInMinutes": null
}
```
</details>

<details>
<summary>Deploy stack with stream enabled</summary>

When `--stream` is enabled, stack events are sent to `stderr`, therefore describe stack output still can be captured by from `stdout`

```bash
gocfn deploy --name hello --parameter-overrides "BucketName=helloza1" --template-file stack.yml --stream 1> output.json
{
    "ClientRequestToken": null,
    "EventId": "ef096ff0-2ffb-11e8-94a1-50a68a2012f2",
    "LogicalResourceId": "hello",
    "PhysicalResourceId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "ResourceProperties": null,
    "ResourceStatus": "UPDATE_IN_PROGRESS",
    "ResourceStatusReason": "User Initiated",
    "ResourceType": "AWS::CloudFormation::Stack",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:12:52.843Z"
}
{
    "ClientRequestToken": null,
    "EventId": "S3Bucket-UPDATE_IN_PROGRESS-2018-03-25T07:12:57.644Z",
    "LogicalResourceId": "S3Bucket",
    "PhysicalResourceId": "helloza",
    "ResourceProperties": "{\"BucketName\":\"helloza1\",\"WebsiteConfiguration\":{\"IndexDocument\":\"index.html\",\"ErrorDocument\":\"error.html\"}}",
    "ResourceStatus": "UPDATE_IN_PROGRESS",
    "ResourceStatusReason": "Requested update requires the creation of a new physical resource; hence creating one.",
    "ResourceType": "AWS::S3::Bucket",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:12:57.644Z"
}
{
    "ClientRequestToken": null,
    "EventId": "S3Bucket-UPDATE_IN_PROGRESS-2018-03-25T07:12:59.141Z",
    "LogicalResourceId": "S3Bucket",
    "PhysicalResourceId": "helloza1",
    "ResourceProperties": "{\"BucketName\":\"helloza1\",\"WebsiteConfiguration\":{\"IndexDocument\":\"index.html\",\"ErrorDocument\":\"error.html\"}}",
    "ResourceStatus": "UPDATE_IN_PROGRESS",
    "ResourceStatusReason": "Resource creation Initiated",
    "ResourceType": "AWS::S3::Bucket",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:12:59.141Z"
}
{
    "ClientRequestToken": null,
    "EventId": "BucketPolicy-UPDATE_IN_PROGRESS-2018-03-25T07:13:24.377Z",
    "LogicalResourceId": "BucketPolicy",
    "PhysicalResourceId": "hello-BucketPolicy-12DJ4X7RUU313",
    "ResourceProperties": "{\"Bucket\":\"helloza1\",\"PolicyDocument\":{\"Version\":\"2012-10-17\",\"Statement\":[{\"Action\":\"s3:GetObject\",\"Resource\":\"arn:aws:s3:::helloza1/*\",\"Effect\":\"Allow\",\"Principal\":\"*\",\"Sid\":\"PublicReadForGetBucketObjects\"}],\"Id\":\"MyPolicy\"}}",
    "ResourceStatus": "UPDATE_IN_PROGRESS",
    "ResourceStatusReason": "Requested update requires the creation of a new physical resource; hence creating one.",
    "ResourceType": "AWS::S3::BucketPolicy",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:13:24.377Z"
}
{
    "ClientRequestToken": null,
    "EventId": "S3Bucket-UPDATE_COMPLETE-2018-03-25T07:13:19.759Z",
    "LogicalResourceId": "S3Bucket",
    "PhysicalResourceId": "helloza1",
    "ResourceProperties": "{\"BucketName\":\"helloza1\",\"WebsiteConfiguration\":{\"IndexDocument\":\"index.html\",\"ErrorDocument\":\"error.html\"}}",
    "ResourceStatus": "UPDATE_COMPLETE",
    "ResourceStatusReason": null,
    "ResourceType": "AWS::S3::Bucket",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:13:19.759Z"
}
{
    "ClientRequestToken": null,
    "EventId": "0430a100-2ffc-11e8-9f2c-503aca41a061",
    "LogicalResourceId": "hello",
    "PhysicalResourceId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "ResourceProperties": null,
    "ResourceStatus": "UPDATE_COMPLETE_CLEANUP_IN_PROGRESS",
    "ResourceStatusReason": null,
    "ResourceType": "AWS::CloudFormation::Stack",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:13:28.315Z"
}
{
    "ClientRequestToken": null,
    "EventId": "BucketPolicy-UPDATE_COMPLETE-2018-03-25T07:13:25.970Z",
    "LogicalResourceId": "BucketPolicy",
    "PhysicalResourceId": "hello-BucketPolicy-TR1WBFLASSSN",
    "ResourceProperties": "{\"Bucket\":\"helloza1\",\"PolicyDocument\":{\"Version\":\"2012-10-17\",\"Statement\":[{\"Action\":\"s3:GetObject\",\"Resource\":\"arn:aws:s3:::helloza1/*\",\"Effect\":\"Allow\",\"Principal\":\"*\",\"Sid\":\"PublicReadForGetBucketObjects\"}],\"Id\":\"MyPolicy\"}}",
    "ResourceStatus": "UPDATE_COMPLETE",
    "ResourceStatusReason": null,
    "ResourceType": "AWS::S3::BucketPolicy",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:13:25.97Z"
}
{
    "ClientRequestToken": null,
    "EventId": "BucketPolicy-UPDATE_IN_PROGRESS-2018-03-25T07:13:25.655Z",
    "LogicalResourceId": "BucketPolicy",
    "PhysicalResourceId": "hello-BucketPolicy-TR1WBFLASSSN",
    "ResourceProperties": "{\"Bucket\":\"helloza1\",\"PolicyDocument\":{\"Version\":\"2012-10-17\",\"Statement\":[{\"Action\":\"s3:GetObject\",\"Resource\":\"arn:aws:s3:::helloza1/*\",\"Effect\":\"Allow\",\"Principal\":\"*\",\"Sid\":\"PublicReadForGetBucketObjects\"}],\"Id\":\"MyPolicy\"}}",
    "ResourceStatus": "UPDATE_IN_PROGRESS",
    "ResourceStatusReason": "Resource creation Initiated",
    "ResourceType": "AWS::S3::BucketPolicy",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:13:25.655Z"
}
{
    "ClientRequestToken": null,
    "EventId": "06a76d60-2ffc-11e8-86b5-50a68a20122a",
    "LogicalResourceId": "hello",
    "PhysicalResourceId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "ResourceProperties": null,
    "ResourceStatus": "UPDATE_COMPLETE",
    "ResourceStatusReason": null,
    "ResourceType": "AWS::CloudFormation::Stack",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:13:32.455Z"
}
{
    "ClientRequestToken": null,
    "EventId": "S3Bucket-a38903d9-cb1c-47d4-a3fa-a6cfc114fc44",
    "LogicalResourceId": "S3Bucket",
    "PhysicalResourceId": "helloza",
    "ResourceProperties": null,
    "ResourceStatus": "DELETE_COMPLETE",
    "ResourceStatusReason": null,
    "ResourceType": "AWS::S3::Bucket",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:13:32.122Z"
}
{
    "ClientRequestToken": null,
    "EventId": "S3Bucket-e142036b-4b00-40d2-85c2-7329f35661ab",
    "LogicalResourceId": "S3Bucket",
    "PhysicalResourceId": "helloza",
    "ResourceProperties": null,
    "ResourceStatus": "DELETE_IN_PROGRESS",
    "ResourceStatusReason": null,
    "ResourceType": "AWS::S3::Bucket",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:13:31.333Z"
}
{
    "ClientRequestToken": null,
    "EventId": "BucketPolicy-e26c9aab-09b3-404c-a800-4a83b8b5576e",
    "LogicalResourceId": "BucketPolicy",
    "PhysicalResourceId": "hello-BucketPolicy-12DJ4X7RUU313",
    "ResourceProperties": null,
    "ResourceStatus": "DELETE_COMPLETE",
    "ResourceStatusReason": null,
    "ResourceType": "AWS::S3::BucketPolicy",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:13:30.337Z"
}
{
    "ClientRequestToken": null,
    "EventId": "BucketPolicy-fea4ce56-3e74-445a-bd4f-dcec5051372f",
    "LogicalResourceId": "BucketPolicy",
    "PhysicalResourceId": "hello-BucketPolicy-12DJ4X7RUU313",
    "ResourceProperties": null,
    "ResourceStatus": "DELETE_IN_PROGRESS",
    "ResourceStatusReason": null,
    "ResourceType": "AWS::S3::BucketPolicy",
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "Timestamp": "2018-03-25T07:13:29.792Z"
}
```
stack information

```bash
cat output.json
{
    "Capabilities": null,
    "ChangeSetId": "arn:aws:cloudformation:us-west-2:111111111111:changeSet/gocfn-cloudformation-package-deploy-1521961936/2341ee88-e4ab-4e06-8acf-1251001ffdd8",
    "CreationTime": "2018-03-01T08:57:19.888Z",
    "DeletionTime": null,
    "Description": null,
    "DisableRollback": false,
    "EnableTerminationProtection": false,
    "LastUpdatedTime": "2018-03-25T07:12:52.843Z",
    "NotificationARNs": null,
    "Outputs": [
        {
            "Description": "Name of S3 bucket to hold website content",
            "ExportName": null,
            "OutputKey": "S3BucketSecureURL",
            "OutputValue": "https://helloza1.s3.amazonaws.com"
        },
        {
            "Description": "URL for website hosted on S3",
            "ExportName": null,
            "OutputKey": "WebsiteURL",
            "OutputValue": "http://helloza1.s3-website-us-west-2.amazonaws.com"
        }
    ],
    "Parameters": [
        {
            "ParameterKey": "BucketName",
            "ParameterValue": "helloza1",
            "ResolvedValue": null,
            "UsePreviousValue": null
        }
    ],
    "ParentId": null,
    "RoleARN": null,
    "RollbackConfiguration": {
        "MonitoringTimeInMinutes": null,
        "RollbackTriggers": null
    },
    "RootId": null,
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "StackStatus": "UPDATE_COMPLETE",
    "StackStatusReason": null,
    "Tags": null,
    "TimeoutInMinutes": null
}
```
</section>

<section>
<summary>Deploy stack with debugging enabled</summary>

gocfn provides extensive debugging information when `--debug` option is given.

```bash
$ gocfn deploy --name hello --parameter-overrides "BucketName=helloza123" --template-file stack.yml --debug

{"level":"debug","msg":"Checking if stack exists","stackName":"hello","time":"2018-03-25T18:20:08+11:00"}
{"level":"debug","msg":"Stack exist with status UPDATE_COMPLETE","stackName":"hello","time":"2018-03-25T18:20:10+11:00"}
{"level":"debug","msg":"Running CreateChangeSet","stackName":"hello","time":"2018-03-25T18:20:10+11:00"}
{"level":"debug","msg":"Waiting for changeset to finish","stackName":"hello","time":"2018-03-25T18:20:10+11:00"}
{"level":"debug","msg":"Running ExecuteChangeSet","stackName":"hello","time":"2018-03-25T18:20:42+11:00"}
{"level":"debug","msg":"Waiting for stack to be created/updated","stackName":"hello","time":"2018-03-25T18:20:43+11:00"}
{"level":"debug","msg":"Stack is ready and no streaming is required","stackName":"hello","time":"2018-03-25T18:21:45+11:00"}
{
    "Capabilities": null,
    "ChangeSetId": "arn:aws:cloudformation:us-west-2:111111111111:changeSet/gocfn-cloudformation-package-deploy-1521962408/4c820a55-85dc-47a8-86a9-7eb116c131ee",
    "CreationTime": "2018-03-01T08:57:19.888Z",
    "DeletionTime": null,
    "Description": null,
    "DisableRollback": false,
    "EnableTerminationProtection": false,
    "LastUpdatedTime": "2018-03-25T07:20:42.994Z",
    "NotificationARNs": null,
    "Outputs": [
        {
            "Description": "Name of S3 bucket to hold website content",
            "ExportName": null,
            "OutputKey": "S3BucketSecureURL",
            "OutputValue": "https://helloza123.s3.amazonaws.com"
        },
        {
            "Description": "URL for website hosted on S3",
            "ExportName": null,
            "OutputKey": "WebsiteURL",
            "OutputValue": "http://helloza123.s3-website-us-west-2.amazonaws.com"
        }
    ],
    "Parameters": [
        {
            "ParameterKey": "BucketName",
            "ParameterValue": "helloza123",
            "ResolvedValue": null,
            "UsePreviousValue": null
        }
    ],
    "ParentId": null,
    "RoleARN": null,
    "RollbackConfiguration": {
        "MonitoringTimeInMinutes": null,
        "RollbackTriggers": null
    },
    "RootId": null,
    "StackId": "arn:aws:cloudformation:us-west-2:111111111111:stack/hello/8978e0f0-1d2e-11e8-a95e-503aca41a0c5",
    "StackName": "hello",
    "StackStatus": "UPDATE_COMPLETE",
    "StackStatusReason": null,
    "Tags": null,
    "TimeoutInMinutes": null
}
```

