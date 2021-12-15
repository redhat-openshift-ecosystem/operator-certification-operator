# operator-certification-operator
A Kubernetes operator to provision resources for the operator certification pipeline. This operator is installed in all
namespaces which can support multi-tenant scenarios. **Note:** This operator should only be used by Red Hat partners attempting to certify their operator(s).

# Requirements
The certification operator requires that you have the following tools installed, functional, and in your path.
- [Install](https://docs.openshift.com/container-platform/4.8/cli_reference/openshift_cli/getting-started-cli.html#installing-openshift-cli) oc, the OpenShift CLI tool (tested with version 4.7.13)
- [Install](https://tekton.dev/docs/cli/) tkn, the Tekton CLI tool (tested with version 0.19.1)
- [Install](https://git-scm.com/downloads) git, the Git CLI tool (tested with 2.32.0)
- The certification pipeline expects you to have the source files of your Operator bundle

# Pre - Installation
The below steps exist for simplicity and to make the installation clearer.
The operator watches all namespaces, so if secrets/configs/etc already exist in another namespace, feel free to use the existing
namespace when following the operator installation steps.

### Create a new namespace where the following secrets will be applied.
`oc new-project oco`

### Add the kubeconfig secret which will be used to deploy the operator under test and run the certification checks.
* Open a terminal window
* Set the `KUBECONFIG` environment variable
```
export KUBECONFIG=/path/to/your/cluster/kubeconfig
```
> *This kubeconfig will be used to deploy the Operator under test and run the certification checks.*
```
oc create secret generic kubeconfig --from-file=kubeconfig=$KUBECONFIG
```
### Configuring steps for submitting the results
- Add the github API token to the repo where the PR will be created
```
oc create secret generic github-api-token --from-literal GITHUB_TOKEN=<github token>
```
- Add RedHat Container API access key
  
  This API access key is specifically related to your unique partner account for Red Hat Connect portal. Instructions to obtain your API key can be found: [here](https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/operator-cert-workflow.md#step-b---get-api-key)
```
oc create secret generic pyxis-api-secret --from-literal pyxis_api_key=< API KEY >
```

- Optional pipeline configurations can be found [here](https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/ci-pipeline.md#optional-configuration)

# Installation Steps
Since this operator isn't in OperatorHub, the process to get it into a cluster is manual at this point.
Please follow the below steps to get the operator into your cluster. Follow the steps from [here](docs/INSTALLATION.md)


# Execute the Pipeline (Development Iterations)
A pre-requisite to running a pipeline is that a `workspace-template.yaml` exists in the directory you want to execute the `tkn` commands from.

To create a workspace-template.yaml
```
cat <<EOF > workspace-template.yaml
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
EOF
```

There are multiple ways to execute the Pipeline which can be found [here](https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/ci-pipeline.md#execute-the-pipeline-development-iterations)
