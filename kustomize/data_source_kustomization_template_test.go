package kustomize

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
)

var resourceTemplate = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: test-basic
spec:
  replicas: 1
  strategy: {}
  template:
    metadata:
      creationTimestamp: null
    spec:
      containers:
      - image: nginx
        name: nginx
        resources: {}
status: {}
`

func TestAccDataSourceKustomizationTemplate_basic(t *testing.T) {
	resource1 := fmt.Sprintf(resourceTemplate, "test1")
	resource2, _ := fromYaml(fmt.Sprintf(resourceTemplate, "test2"))

	kustomization := map[interface{}]interface{}{
		"bases":     []string{"../test_kustomizations/template"},
		"resources": []interface{}{resource1, resource2},
	}
	config := testAccDataSourceKustomizationTemplateConfig_basic(kustomization)
	resource.Test(t, resource.TestCase{
		//PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.kustomization_template.test", "id"),
					resource.TestCheckResourceAttrSet("data.kustomization_template.test", "kustomization"),
					resource.TestCheckResourceAttr("data.kustomization_template.test", "ids.#", "6"),
					resource.TestCheckResourceAttr("data.kustomization_template.test", "manifests.%", "6"),
				),
			},
		},
	})
}

func testAccDataSourceKustomizationTemplateConfig_basic(kustomization map[interface{}]interface{}) string {
	kustomizationYaml, _ := toYaml(kustomization)
	return fmt.Sprintf(`
data "kustomization_template" "test" {
	kustomization = <<-EOF
%s
EOF
}
`, string(kustomizationYaml))
}
