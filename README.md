# redshift-udf-awscli

A PoC for flexible access to multi-AWS services from Redshift.

## How to run.

### Container image build and push.

with docker login to AWS.
```console 
$ make build
$ make push 
``` 

### Deploy Lambda Function with Container Image.

For example: with [github.com/fujiwara/lambroll](https://github.com/fujiwara/lambroll)

funcion.json is following:
```json
{
  "FunctionName": "redshift-udf-awscli",
  "MemorySize": 128,
  "Role": "arn:aws:iam::012345678912:role/redshift-udf-awscli",
  "PackageType": "Image",
  "Code": {
    "ImageUri": "012345678912.dkr.ecr.ap-northeast-1.amazonaws.com/lambda/redshift-udf-awscli:latest"
  }
}
```

```console
$ lambroll deploy
```

### Create Redshift UDF

