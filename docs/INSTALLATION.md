### Create a CatalogSource
```
oc apply -f - <<'EOF'
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: operator-certification-operator
  namespace: openshift-marketplace
spec:
  displayName: Operator Certification Operator
  icon:
    base64data: ""
    mediatype: ""
  image: quay.io/opdev/operator-certification-operator-index:latest
  priority: -200
  publisher: Red Hat
  sourceType: grpc
  updateStrategy:
    registryPoll:
      interval: 10m0s
EOF
```

### Installing Operator Certification Operator
* Use the left-hand menu to navigate to *Operators*
* In the *Operators* submenu click on *OperatorHub*
* Use the Filter/Search box to filter on *Operator Certification Operator*
* Click the *Operator Certification Operators* tile
* In the flyout menu to the right click the *Install* button near the top
* On the next screen "Install Operator" scroll to the bottom of the page and click *Install*
* Click the *View Operator* button

### Applying an Operator Pipeline Custom Resource
* In the *Project dropdown* select the *Project* you wish to apply the Custom Resource
* Under *OP: Operator Pipeline* click *Create instance*
* The *Create OperatorPipeline* screen should be pre-populated with default values
  * If the pre-installation steps were followed and all resource names are the same, nothing should need to be changed
* Click *Create*
* The CR will get created and the Operator will start reconciling

### Check the Conditions of the Custom Resource
* Click on the name of the Custom Resource you created above *operatorpipeline-sample*
* Scroll down to the *Conditions* section
* Validate that all *Status* values are *True*
  * If a resource fails reconciliation the *Message* section should indicate what needs correction
  
### Optionally Check the Operator Logs
* `oc get pods -n openshift-marketplace`
* Copy the full pod name of the `certification-operator-controller-manager` pod
* `oc get logs -f -n openshift-marketplace <pod name> manager`
* Check to see if the reconciliation occurred 

## Uninstalling the Operator Pipeline Custom Resource
* From the *Operator Certification Operator* main page 
* Click *Operator Pipeline* in the display bar
* Click the three dots on the right for the Custom Resource
* Select *Uninstall*

## Uninstalling the Operator
* Navigate to *Installed Operators* 
* Search for the *Operator*
* Click the three dots on the right
* Select *Uninstall*

## Uninstalling a CatalogSource
`oc delete catalogsource -n openshift-marketplace operator-certification-operator`
