# Developer Documentation

## Requirements

Development and testing the operator requires that you have the following tools installed,
functional, and in your path.

| Name             | Tool cli          | Minimum version |
|----------------- |:-----------------:| ---------------:|
| OperatorSDK      | `operator-sdk`    | v1.15.0         |
| OpenShift Client | `oc`              | v4.7.13         |


### Prerequisite - Install OpenShift Pipelines Operator
This operator has the OpenShift Pipelines Operator as a dependency, which OLM manages when installing via a catalog. However,
for local development, this dependency must be installed before *Testing Locally* can occur.
* Log into your cluster's OpenShift Console with cluster admin privileges
* Use the left-hand menu to navigate to *Operators*
* In the *Operators* submenu click on *OperatorHub*
* Use the Filter/Search box to filter on *OpenShift Pipelines*
* Click the *Red Hat OpenShift Pipelines* tile
* In the flyout menu to the right click the *Install* button near the top
* On the next screen "Install Operator" scroll to the bottom of the page and click *Install*


## Testing Locally
1. Have a cluster up and running
2. Run `make install` to install `CRD's`
3. Run `make run` to start the operator
   1. Or start the operator in your preferred manner
4. Run `./docs/dev/seed.sh` to see all the configs/secrets in the cluster
   1. Depending on what reconciler you are working on feel free to comment out anything in the file not related