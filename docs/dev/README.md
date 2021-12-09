# Developer Documentation

## Requirements

Development and testing the operator requires that you have the following tools installed,
functional, and in your path.

| Name             | Tool cli          | Minimum version |
|----------------- |:-----------------:| ---------------:|
| OperatorSDK      | `operator-sdk`    | v1.15.0         |
| OpenShift Client | `oc`              | v4.7.13         |


## Testing Locally
1. Have a cluster up and running
2. Run `make install` to install `CRD's`
3. Run `make run` to start the operator
   1. Or start the operator in your preferred manner
4. Run `./docs/dev/seed.sh` to see all the configs/secrets in the cluster
   1. Depending on what reconciler you are working on feel free to comment out anything in the file not related