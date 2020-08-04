package kustomize

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

var fileArrayFields = [...]string {
	"configurations",
	"patchesStrategicMerge",
	"resources",
}

func dataSourceKustomizationTemplate() *schema.Resource {
	return &schema.Resource{
		Read: kustomizationTemplateBuild,
		Schema: map[string]*schema.Schema{
			"kustomization": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Kustomization yaml as map.  See https://kubectl.docs.kubernetes.io/pages/reference/kustomize.html",
			},
			"ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"manifests": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func kustomizationTemplateBuild(d *schema.ResourceData, m interface{}) error {
	overlay, err := MakefsOverlay(); if err != nil {
		return err
	}
	kustomization, err := fromYaml(d.Get("kustomization").(string)); if err != nil {
		return  err
	}

	for _, field := range fileArrayFields {
		err := addFileListToKustomize(overlay, kustomization, field); if err != nil {
			return err
		}
	}
	err = addPatchesjson6902ToKustomize(overlay, kustomization); if err != nil {
		return err
	}

	kustomizationYaml, err := toYaml(kustomization); if err != nil {
		return err
	}
	err = overlay.AddOverlayFile("kustomization.yaml", kustomizationYaml); if err != nil {
		return err
	}
	err = setResourcesFromKustomizeUsingFs(d, overlay, overlay.rootDir)

	return err
}

func addFileListToKustomize(overlay FsOverlay, kustomization map[interface{}]interface{}, key string) error {
	var specs []interface{}
	value, ok := kustomization[key]; if !ok {
		return nil
	} else {
		specs = value.([]interface{})
	}

	if len(specs) == 0 {
		delete(kustomization, key)
		return nil
	}

	names, err := overlay.AddOverlayFiles(key, specs); if err != nil {
		return err
	}

	kustomization[key] = names
	return nil
}

func addPatchesjson6902ToKustomize(overlay FsOverlay, kustomization map[interface{}]interface{}) error {
	key := "patchesJson6902"
	field := "path"

	var specs []map[string]interface{}
	value, ok := kustomization[key]; if !ok {
		return nil
	} else {
		specs = value.([]map[string]interface{})
	}

	if len(specs) == 0 {
		delete(kustomization, key)
		return nil
	}

	paths := make([]interface{}, len(specs))
	for ix, patch := range specs {
		paths[ix], ok = patch[field]; if !ok {
			return fmt.Errorf("%s does not contain %s (%v)", key, field, patch)
		}
	}

	names, err := overlay.AddOverlayFiles(key, paths); if err != nil {
		return err
	}
	for ix, patch := range specs {
		patch[field] = names[ix]
	}
	return nil
}
