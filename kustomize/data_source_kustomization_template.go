package kustomize

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/otiai10/copy"
)

var basesTemplate = `
bases:
  - ./bases
`

var resourcesTemplate = `
resources:
  - ./resources.yaml
`

var patchesTemplate = `
patchesStrategicMerge:
  - ./patches.yaml
`

func dataSourceKustomizationTemplate() *schema.Resource {
	return &schema.Resource{
		Read: kustomizationTemplateBuild,

		Schema: map[string]*schema.Schema{
			"bases_path": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"kustomization": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"patches": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"resources": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"ids": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"manifests": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func kustomizationTemplateBuild(d *schema.ResourceData, m interface{}) error {
	basesPath := d.Get("bases_path").(string)
	kustomization := d.Get("kustomization").(string)
	patches := GetStringList(d, "patches")
	resources := GetStringList(d,"resources")
	tempDir, err := ioutil.TempDir("", "kustomizationTemplateBuild")
	if err != nil {
		return fmt.Errorf("kustomizationTemplateBuild: %s", err)
	}
	err = kustomizationTemplateMerge(tempDir, basesPath, kustomization, patches, resources)
	if err != nil {
		return fmt.Errorf("kustomizationTemplateBuild: %s", err)
	}

	rm, err := runKustomizeBuild(tempDir)
	if err != nil {
		return fmt.Errorf("kustomizationTemplateBuild: %s", err)
	}

	_ = d.Set("ids", flattenKustomizationIDs(rm))

	outResources, err := flattenKustomizationResources(rm)
	if err != nil {
		return fmt.Errorf("kustomizationTemplateBuild: %s", err)
	}
	_ = d.Set("manifests", outResources)

	id, err := getIDFromResources(rm)
	if err != nil {
		return fmt.Errorf("kustomizationTemplateBuild: %s", err)
	}
	d.SetId(id)
	_ = os.RemoveAll(tempDir)

	return nil
}

func kustomizationTemplateMerge(tempPath string, basesPath string, kustomization string, patches []string, resources []string) error {
	copy.Copy(basesPath, filepath.Join(tempPath, "bases"))
	err := writeKustomization(filepath.Join(tempPath, "kustomization.yaml"), kustomization, len(patches) > 0, len(resources) > 0)
	if err != nil {
		return fmt.Errorf("kustomizationTemplateMerge: %s", err)
	}
	err = writeYamlArray(filepath.Join(tempPath, "patches.yaml"), patches)
	if err != nil {
		return fmt.Errorf("kustomizationTemplateMerge: %s", err)
	}
	err = writeYamlArray(filepath.Join(tempPath, "resources.yaml"), resources)
	if err != nil {
		return fmt.Errorf("kustomizationTemplateMerge: %s", err)
	}

	return nil
}

func writeKustomization(filePath string, kustomization string, hasPatches bool, hasResources bool) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("writeKustomization: %s", err)
	}
	_, err = f.WriteString(basesTemplate)
	if err != nil {
		return fmt.Errorf("writeKustomization: %s", err)
	}
	if hasPatches {
		_, err = f.WriteString(patchesTemplate)
		if err != nil {
			return fmt.Errorf("writeKustomization: %s", err)
		}
	}
	if hasResources {
		_, err = f.WriteString(resourcesTemplate)
		if err != nil {
			return fmt.Errorf("writeKustomization: %s", err)
		}
	}
	_, err = f.WriteString(kustomization)
	if err != nil {
		return fmt.Errorf("writeKustomization: %s", err)
	}
	err = f.Close()
	if err != nil {
		return fmt.Errorf("writeKustomization: %s", err)
	}
	return nil
}

func writeYamlArray(filePath string, arg []string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("writeYamlArray: %s", err)
	}
	for _, spec := range arg {
		_, err = f.WriteString("---\n")
		if err != nil {
			return fmt.Errorf("writeYamlArray: %s", err)
		}
		_, err = f.WriteString(spec)
		if err != nil {
			return fmt.Errorf("writeYamlArray: %s", err)
		}
		_, err = f.WriteString("\n")
		if err != nil {
			return fmt.Errorf("writeYamlArray: %s", err)
		}
	}
	err = f.Close()
	if err != nil {
		return fmt.Errorf("writeYamlArray: %s", err)
	}
	return nil
}

func GetStringList(d *schema.ResourceData, key string) []string {
	itemsRaw := d.Get(key).([]interface{})
	items := make([]string, len(itemsRaw))
	for i, raw := range itemsRaw {
		items[i] = raw.(string)
	}
	return items
}
