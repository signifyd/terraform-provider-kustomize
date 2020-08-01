package kustomize

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

var resources = []string{
`<<-EOF
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
`,
}

func TestAccDataSourceKustomizationTemplate_basic(t *testing.T) {
	config := testAccDataSourceKustomizationTemplateConfig_basic("../test_kustomizations/template", "",
		resources, []string{})
	resource.Test(t, resource.TestCase{
		//PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.kustomization_template.test", "id"),
					resource.TestCheckResourceAttrSet("data.kustomization_template.test", "bases_path"),
					resource.TestCheckResourceAttr("data.kustomization_template.test", "bases_path", "../test_kustomizations/template"),
					resource.TestCheckResourceAttr("data.kustomization_template.test", "ids.#", "5"),
					resource.TestCheckResourceAttr("data.kustomization_template.test", "manifests.%", "5"),
				),
			},
		},
	})
}

func testAccDataSourceKustomizationTemplateConfig_basic(basesPath string, kustomization string, resources []string, pathches []string) string {
	return fmt.Sprintf(`
data "kustomization_template" "test" {
	bases_path = "%s"
	kustomization = "%s"
	resources = %s
	patches = %s
}
`, basesPath, kustomization, quoteArray(resources), quoteArray(pathches))
}

func quoteArray(arg []string) string {
	return fmt.Sprintf("[%s]", strings.Join(arg, ","))
}