echo switching to openshift-operators namespace
oc project openshift-operators

echo exporting kubeconfig
export KUBECONFIG=~/.kube/config

echo creating kube secret in cluster
oc create secret generic kubeconfig --from-file=kubeconfig=$KUBECONFIG

echo creating github secret in cluster
oc create secret generic github-api-token --from-literal GITHUB_TOKEN=123456789

#echo creating pyxis api key secret in cluster
oc create secret generic pyxis-api-secret --from-literal pyxis_api_key=8675309

echo creating github ssh credentials secret in cluster
oc apply -f - <<'EOF'
kind: Secret
apiVersion: v1
metadata:
  name: github-ssh-credentials
data:
  id_rsa: |
        MTIzNDU2Nzg5
EOF

echo creating docker-registry secret in cluster
oc create secret docker-registry registry-dockerconfig-secret \
    --docker-server=quay.io \
    --docker-username=test \
    --docker-password=123456 \
    --docker-email=test@test.net

echo creating OperatorPipeline
oc apply -f - <<'EOF'
apiVersion: certification.redhat.com/v1alpha1
kind: OperatorPipeline
metadata:
  name: operatorpipeline-sample
spec:
  operatorPipelinesRelease: main
  kubeconfigSecretName: "kubeconfig"
  gitHubSecretName: "github-api-token"
  pyxisSecretName: "pyxis-api-secret"
  dockerRegistrySecretName: "registry-dockerconfig-secret"
  githubSSHSecretName: "github-ssh-credentials"
EOF
