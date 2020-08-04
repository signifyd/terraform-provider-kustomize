package kustomize

import (
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/filesys"
	fLdr "sigs.k8s.io/kustomize/api/loader"

)

var _ filesys.FileSystem = FsOverlay{}

// FsOverlay implements FileSystem using the local filesystem.
type FsOverlay struct {
	base filesys.FileSystem
	overlay filesys.FileSystem
}

// MakefsOverlay makes an instance of FsOverlay.
func MakefsOverlay() FsOverlay {

	return FsOverlay{
		overlay: filesys.MakeEmptyDirInMemory(),
		base: filesys.MakeFsOnDisk(),
	}
}

// Create delegates to os.Create.
func (fs FsOverlay) Create(name string) (filesys.File, error) {
	return fs.base.Create(name)
}

// Mkdir delegates to os.Mkdir.
func (fs FsOverlay) Mkdir(name string) error {
	return fs.base.Mkdir(name)
}

// MkdirAll delegates to os.MkdirAll.
func (fs FsOverlay) MkdirAll(name string) error {
	return fs.base.MkdirAll(name)
}

// RemoveAll delegates to os.RemoveAll.
func (fs FsOverlay) RemoveAll(name string) error {
	return fs.base.RemoveAll(name)
}

// Open delegates to os.Open.
func (fs FsOverlay) Open(name string) (filesys.File, error) {
	f, err := fs.overlay.Open(name); if err == nil {
		return f, nil
	}
	return fs.base.Open(name)
}

// CleanedAbs converts the given path into a
// directory and a file name, where the directory
// is represented as a ConfirmedDir and all that implies.
// If the entire path is a directory, the file component
// is an empty string.
func (fs FsOverlay) CleanedAbs(path string) (filesys.ConfirmedDir, string, error) {
	if fs.overlay.Exists(path) {
		return fs.overlay.CleanedAbs(path)
	}
	return fs.base.CleanedAbs(path)
}

// Exists returns true if os.Stat succeeds.
func (fs FsOverlay) Exists(name string) bool {
	return fs.overlay.Exists(name) || fs.base.Exists(name)
}

// Glob returns the list of matching files
func (fs FsOverlay) Glob(pattern string) ([]string, error) {
	var m []string
	resOverlay, errOverlay := fs.overlay.Glob(pattern); if errOverlay == nil {
		m = append(m, resOverlay...)
	}
	resBase, errBase := fs.base.Glob(pattern); if errBase == nil {
		m = append(m, resBase...)
	}
	return m, nil
}

// IsDir delegates to os.Stat and FileInfo.IsDir
func (fs FsOverlay) IsDir(name string) bool {
	return fs.base.IsDir(name)
}

// ReadFile delegates to ioutil.ReadFile.
func (fs FsOverlay) ReadFile(name string) ([]byte, error) {
	if fs.overlay.Exists(name) {
		return fs.overlay.ReadFile(name)
	}
	return fs.base.ReadFile(name)
}

// WriteFile delegates to ioutil.WriteFile with read/write permissions.
func (fs FsOverlay) WriteFile(name string, c []byte) error {
	return fs.base.WriteFile(name, c)
}

// Walk delegates to filepath.Walk.
func (fs FsOverlay) Walk(path string, walkFn filepath.WalkFunc) error {
	_ = fs.overlay.Walk(path, walkFn)
	_ = fs.overlay.Walk(path, walkFn)
	return nil
}

func (fs FsOverlay) AddOverlayFiles(prefix string, specOrNames []interface{}) ([]string, error) {
	ldr, err := fLdr.NewLoader(fLdr.RestrictionRootOnly, ".", fs.base)
	if err != nil {
		return nil, err
	}
	defer ldr.Cleanup()

	names := make([]string, len(specOrNames))

	for ix, specOrName := range specOrNames {
		name := fmt.Sprintf("%s_%d", prefix, ix)
		switch specOrName.(type) {
		case string:
			specOrNameStr := specOrName.(string)
			_, loadErr := ldr.New(specOrNameStr)

			// If kustomize can load than it is a valid file else treat as data
			if loadErr == nil {
				names[ix] = specOrNameStr
			} else {
				names[ix] = name
				err = fs.AddOverlayFile(name, []byte(specOrNameStr)); if err != nil {
					return names, err
				}
			}
			break
		case map[interface{}]interface{}:
			spec, err := toYaml(specOrName.(map[interface{}]interface{})); if err != nil {
				return names, err
			}
			names[ix] = name
			err = fs.AddOverlayFile(name, spec); if err != nil {
				return names, err
			}
			break
		default:
			return names, fmt.Errorf("unsupported type: %T", specOrName)
		}
	}
	return names, nil
}

func (fs FsOverlay) AddOverlayFile(name string, data []byte) error {
	return fs.overlay.WriteFile(name, data)
}

func toYaml(data map[interface{}]interface{}) ([]byte, error) {
	yamlBytes, err := yaml.Marshal(&data)
	if err != nil {
		return nil, err
	}
	return  yamlBytes, nil
}

func fromYaml(yamlStr string) (map[interface{}]interface{}, error) {
	if yamlStr == "" {
		return nil, nil
	}

	t := make(map[interface{}]interface{})

	err := yaml.Unmarshal([]byte(yamlStr), &t)
	if err != nil {
		return nil, fmt.Errorf("Invalid yaml:\n%s\n%s", yamlStr, err)
	}
	return t, nil
}

func validateYamlString(yamlStr string) ([]byte, error) {
	if yamlStr == "" {
		return nil, nil
	}

	t := make(map[string]interface{})

	err := yaml.Unmarshal([]byte(yamlStr), &t)
	if err != nil {
		return nil, fmt.Errorf("Invalid yaml:\n%s\n%s", yamlStr, err)
	}
	yamlBytes, err := yaml.Marshal(&t)
	if err != nil {
		return nil, err
	}
	return  yamlBytes, nil
}
