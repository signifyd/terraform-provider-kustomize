# Terraform Provider Kustomize

![Run Tests](https://github.com/kbst/terraform-provider-kustomize/workflows/Run%20Tests/badge.svg?branch=master&event=push)

This provider aims to solve 3 common issues of applying a kustomization using kubectl by integrating Kustomize and Terraform.

1. Lack of feedback what changes will be applied.
1. Resources from a previous apply not in the current apply are not purged.
1. Immutable changes like e.g. changing a deployment's selector cause the apply to fail mid way.

To solve this the provider uses the Terraform state to show changes to each resource individually during plan as well as track resources in need of purging.

It also uses [server side dry runs](https://kubernetes.io/docs/reference/using-api/api-concepts/#dry-run) to validate changes to the desired state and translate this into a Terraform plan that will show if a resource will be updated in-place or requires a delete and recreate to apply the changes.

As such it can be useful both to replace kustomize/kubectl integrated into a Terraform configuration as a provisioner as well as standalone `kubectl diff/apply` steps in CI/CD.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) 0.12.x
- [Go](https://golang.org/doc/install) 1.13 (to build the provider plugin)

## Using the template form
This approach uses hcl to build the kustomization.yaml file.  
See https://kubectl.docs.kubernetes.io/pages/reference/kustomize.html for reference.

Wherever the reference specifies that a file path is used the template allows a map
or a yaml string instead and creates an in memory file with the yaml content.  Where the field
is a list, e.g. `bases` real file paths and yaml can be intermixed

Using `yamlencode` makes it easier to define `kustomization` as an hcl map.

Supported fields for file substitution are:
* configurations
* patchesJson6902
* patchesStrategicMerge
* resources

```hcl

data "kustomization_template" "test" {
    #yamlencode is used to convert the map to a yaml string 
	kustomization = yamlencode({
		bases = ["../test_kustomizations/template"]
		resources = ["./overlays/some_resource.yaml", <<-EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  creationTimestamp: null
  labels:
    app: test2
  name: test2
  namespace: test-basic
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test2
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
      labels:
        app: test2
    spec:
      containers:
      - image: nginx
        name: nginx
        resources: {}
status: {}
EOF
]
	})
}

resource "kustomization_resource" "example" {
  for_each = data.kustomization.example.ids

  manifest = data.kustomization.example.manifests[each.value]
}

```

## Usage

```hcl
data "kustomization" "example" {
  # path to kustomization directory
  path = "test_kustomizations/basic/initial"
}

resource "kustomization_resource" "example" {
  for_each = data.kustomization.example.ids

  manifest = data.kustomization.example.manifests[each.value]
}

```

## Configuring the provider

```hcl
provider "kustomization" {
  # optional path to kubeconfig file
  # falls back to KUBECONFIG or KUBE_CONFIG env var
  # or finally '~/.kube/config'
  kubeconfig_path = "/path/to/kubeconfig/file"

  # optional raw kubeconfig string
  # overwrites kubeconfig_path
  kubeconfig_raw = data.template_file.kubeconfig.rendered

  # optional context to use in kubeconfig with multiple contexts
  # if unspecified, the default (current) context is used
  context = "my-context"
}
```

## State import for kustomization_resource

To import existing Kubernetes resources into the Terraform state for above usage example, use a command like below and replace `apps_v1_Deployment|test-basic|test` accordingly. Please note the single quotes required for most shells.

```
terraform import 'kustomization_resource.test["apps_v1_Deployment|test-basic|test"]' 'apps_v1_Deployment|test-basic|test'
```

## Building and Developing the Provider

To work on the provider, you need go installed on your machine (version 1.13.x tested). The provider uses go mod to manage its dependencies, so GOPATH is not required.

To compile the provider, run `make build` as shown below. This will build the provider and put the provider binary in the `terraform.d/plugins/linux_amd64/` directory.

```sh
$ make build
```

In order to test the provider, you can simply run the acceptance tests using `make test`. You can set the `KUBECONFIG` environment variable to point the tests to a specific cluster or set the context of your current config accordingly. The tests create namespaces on the current context. [Kind](https://github.com/kubernetes-sigs/kind) or [Minikube](https://github.com/kubernetes/minikube) clusters work well for testing.

```sh
$ make test
```
