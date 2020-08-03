package kustomize

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/otiai10/copy"
	"gopkg.in/yaml.v2"
)

var basesTemplate = `
bases:
  - %s
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
			"bases_path": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Kustomization bases path.  Either a local directory, git or http",
			},
			"kustomization": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Additional elements to add to the kustomization.yaml file.  Must be in yaml format.",
			},
			"patches": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Added to the kustomization.yaml file as `patchesStrategicMerge`.   Must be in yaml format.",
			},
			"resources": {
				Type:        schema.TypeList,
				Optional:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Added to the kustomization.yaml file as `resources`.   Must be in yaml format.",
			},
			"ids": {
				Type:        schema.TypeSet,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "IDs of each resource manifest returned.",
			},
			"manifests": {
				Type:        schema.TypeMap,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Description: "Resource manifests returned.",
			},
		},
	}
}

func kustomizationTemplateBuild(d *schema.ResourceData, m interface{}) error {
	basesPath := d.Get("bases_path").(string)
	kustomization, err := validateYamlString(d.Get("kustomization").(string))
	if err != nil {
		return err
	}
	patches, err := validateYamlStringList(GetStringList(d, "patches"))
	if err != nil {
		return err
	}
	resources, err := validateYamlStringList(GetStringList(d, "resources"))
	if err != nil {
		return err
	}

	tempDir, err := ioutil.TempDir("", "kustomizationTemplateBuild")
	if err != nil {
		return fmt.Errorf("kustomizationTemplateBuild: %s", err)
	}
	defer os.RemoveAll(tempDir)

	relBasesPath, err := getRelativeBasesPath(basesPath, tempDir)
	if err != nil {
		return fmt.Errorf("kustomizationTemplateBuild: %s", err)
	}
	err = kustomizationTemplateMerge(tempDir, relBasesPath, kustomization, patches, resources)
	if err != nil {
		return fmt.Errorf("kustomizationTemplateBuild: %s", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("kustomizationTemplateBuild: %s", err)
	}
	_ = os.Chdir(tempDir)
	defer os.Chdir(cwd)
	err = setResourcesFromKustomize(d, tempDir)

	return err
}

func kustomizationTemplateMerge(tempPath string, relBasesPath string, kustomization string, patches []string, resources []string) error {
	err := writeKustomization(filepath.Join(tempPath, "kustomization.yaml"), kustomization, relBasesPath,
		len(patches) > 0, len(resources) > 0)
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

func getRelativeBasesPath(basesPath string, tempDir string) (string, error) {
	if _, err := os.Stat(basesPath); os.IsNotExist(err) {
		// Assume this is a git path
		return basesPath, nil
	}
	err := copy.Copy(basesPath, filepath.Join(tempDir, "bases"))
	if err != nil {
		return "", err
	}
	return "bases", nil
}

func writeKustomization(filePath string, kustomization string, basesPath string, hasPatches bool, hasResources bool) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("writeKustomization: %s", err)
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf(basesTemplate, basesPath))
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
	return nil
}

func writeYamlArray(filePath string, arg []string) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("writeYamlArray: %s", err)
	}
	defer f.Close()

	for _, spec := range arg {
		_, err = f.WriteString("---\n")
		if err != nil {
			return err
		}
		_, err = f.WriteString(spec)
		if err != nil {
			return err
		}
		_, err = f.WriteString("\n")
		if err != nil {
			return err
		}
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

func validateYamlStringList(yamlList []string) ([]string, error) {
	items := make([]string, len(yamlList))
	for i, yamlStr := range yamlList {
		yamlNormalized, err := validateYamlString(yamlStr)
		if err != nil {
			return []string{}, err
		}
		items[i] = yamlNormalized
	}
	return items, nil
}

func validateYamlString(yamlStr string) (string, error) {
	if yamlStr == "" {
		return "", nil
	}

	t := make(map[string]interface{})

	err := yaml.Unmarshal([]byte(yamlStr), &t)
	if err != nil {
		return "", fmt.Errorf("Invalid yaml:\n%s\n%s", yamlStr, err)
	}
	yamlBytes, err := yaml.Marshal(&t)
	if err != nil {
		return "", err
	}
	yamlNormalized := string(yamlBytes)
	return yamlNormalized, nil
}
