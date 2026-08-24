# Developer Documentation

## Dev Setup

Development and testing the operator requires that you have the following tools installed,
functional, and in your path.

| Tool             | CLI            | Minimum Version |
|------------------|:--------------:|----------------:|
| Go               | `go`           |         v1.26.7 |
| Make             | `make`         |               - |
| OperatorSDK      | `operator-sdk` |         v1.40.0 |
| OpenShift Client | `oc`           |         v4.7.13 |
| Tekton CLI       | `tkn`          |         v0.19.1 |
| Git              | `git`          |         v2.32.0 |

**Note:** For Tekton CLI, the version should be based on the [Compatibility and support matrix](https://docs.redhat.com/en/documentation/red_hat_openshift_pipelines/1.23/html/release_notes/index) for the Red Hat OpenShift Pipelines Operator.
This ensures that the version of Tekton CLI used locally, is supported by the installed Pipelines operator within your OpenShift cluster.

### Cluster Requirements

* **OpenShift 4.10+** — CRC works fine for local development
* **Cluster-admin access** — needed to install CRDs and operators
* **OpenShift Pipelines Operator** — installed via OperatorHub (see below)

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
2. Run `make install` to install CRDs
3. Export GIT_REPO_PATH. This path must be writable on your host machine since `make run` executes locally. `export GIT_REPO_PATH=/tmp/git-repo`
4. Run `make run` to start the operator
   1. Or start the operator in your preferred manner
5. Export `KUBECONFIG` for your target cluster and verify `oc` is logged in:
   ```bash
   export KUBECONFIG=/path/to/cluster/kubeconfig
   oc whoami
   ```
6. Run `./docs/dev/seed.sh` to seed all the configs/secrets in the cluster
   1. This creates dummy secrets the operator needs: kubeconfig (to deploy test operators), GitHub token (for PR creation), Pyxis API key (for certification submission), Docker registry credentials, and SSH keys
      1. **Optional** Depending on how you want to execute the CI pipeline, different secrets may be required — see [CI pipeline execution methods](https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/ci-pipeline.md#execute-the-pipeline-development-iterations)
   2. Depending on what reconciler you are working on feel free to comment out anything in the file not related

**Verify it's working:**

**For local development (make run):**
- Check the terminal output where you ran `make run` — you should see reconciliation messages when the OperatorPipeline CR is created
- Check the CR status (ensure you're in the correct project first):
  ```bash
  oc project <your-project>  # or use -n <your-project> flag
  oc get operatorpipeline operatorpipeline-sample -o yaml
  ```
  Conditions should show `True` for successfully reconciled resources

**For installed operator deployment:**

**Using oc:**
- Check operator logs:
  ```bash
  oc logs -n openshift-operators -l control-plane=controller-manager,app.kubernetes.io/name=operator-certification-operator -f
  ```
- You should see reconciliation messages when the OperatorPipeline CR is created
- Check the CR status (ensure you're in the correct project first):
  ```bash
  oc project <your-project>  # or use -n <your-project> flag
  oc get operatorpipeline operatorpipeline-sample -o yaml
  ```
  Conditions should show `True` for successfully reconciled resources

**Using the console:**
- Navigate to *Operators* → *Installed Operators* → *Operator Certification Operator*
- Click on *OperatorPipeline* tab
- Select `operatorpipeline-sample` — the Status column should show reconciliation state
- Click into the resource and scroll to *Conditions* — all conditions should have `Status: True`
- Check operator logs by navigating to *Workloads* → *Pods*, filter by `openshift-operators` namespace, find the `certification-operator-controller-manager` pod, and view logs
