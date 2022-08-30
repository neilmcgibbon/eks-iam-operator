# EKS IAM Operator Helm Chart

Add eks-iam-operator repository to Helm:

```
helm repo add eks-iam-operator  https://neilmcgibbon.github.io/eks-iam-operator
```

Install (or upgrade) chart:

```
# Example installs into kube-system namespace, but any namespace is possible
# Example uses chart name of eks-iam-operator, but any name is possible
helm upgrade --install eks-iam-operator --namespace kube-system eks-iam-operator/eks-iam-operator
```


## Configuration

### Important! 

The following parmagers are required to be overridden in your values file:
 - `serviceAccount.roleArn`
 - `config.oidc.providerArn`
 - `config.oidc.issuerUrl`


| Parameter | Description | Default |
|-|-|-|
| `affinity` | Map of node/pod affinities	 | `{}` | 
| `config.inlinePolicyNameOptions.prefix` | Prefix to prepend to all inline policies created by the controller | `` | 
| `config.inlinePolicyNameOptions.suffix` | Suffix to append to all inline policies created by the controller | `` | 
| `config.oidc.issuerUrl` | EKS OIDC issuer URL | `` | 
| `config.oidc.providerArn` | EKS OIDC provider ARN | `` | 
| `config.roleNameOptions.prefix` | Prefix to prepend to all roles created by the controller | `` | 
| `config.roleNameOptions.suffix` | Suffix to append to all roles created by the controller | `` | 
| `containers.manager.image.repository` | Override the repo used to pull the controller manager image | `ghcr.io/neilmcgibbon/eks-iam-operator` | 
| `containers.manager.image.tag` | Override the image tag of the controller manager image | `<FIXED VERSION>, see values.yaml` | 
| `containers.manager.resources` | Kubernetes resource object of request & limits for controller manager | `{}` |
| `containers.rbacProxy.resources` | Kubernetes resource object of request & limits for RBAC proxy | `{}` |
| `manager.leaderElect` | Whether or not to perform leader election | `true` | 
| `nodeSelector` | Node labels for pod assignment	 | `{}` | 
| `replicaCount` | How many copies of the controller to run concurrently | `1` | 
| `serviceAccount.annotations` | User provided list of annotations to add to service account | `[]` | 
| `serviceAccount.create` | Whether or not to create a service account | `true` | 
| `serviceAccount.labels` | User provided list of labels to add to service account | `[]` |
| `serviceAccount.roleArn` | AWS IAM Role ARN which provides permissions for this controller to perform functions. Please see the main README.md for required permissions | `` | 
| `tolerations` | Optional deployment tolerations	 | `[]` | 
