package catalog

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"

	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/cli/commands/cmdutils"
	"gitlab.com/gitlab-org/cli/pkg/iostreams"
	"gopkg.in/yaml.v3"
)

const (
	publishToCatalogApiPath = "projects/%s/catalog/publish"
	templatesDirName        = "templates"
	templateFileExt         = ".yml"
	templateFileName        = "template.yml"
)

type publishToCatalogRequest struct {
	Version  string                 `json:"version"`
	Metadata map[string]interface{} `json:"metadata"`
}

type publishToCatalogResponse struct {
	CatalogUrl string `json:"catalog_url"`
}

func Publish(io *iostreams.IOStreams, client *gitlab.Client, repoName string, tagName string) error {
	color := io.Color()

	io.Logf("%s Publishing release %s=%s to the GitLab CI/CD catalog for %s=%s...\n",
		color.ProgressIcon(),
		color.Blue("tag"), tagName,
		color.Blue("repo"), repoName)

	body, err := publishToCatalogRequestBody(tagName)
	if err != nil {
		return cmdutils.WrapError(err, "failed to create a request body")
	}

	path := fmt.Sprintf(publishToCatalogApiPath, url.PathEscape(repoName))
	request, err := client.NewRequest(http.MethodPost, path, body, nil)
	if err != nil {
		return cmdutils.WrapError(err, "failed to create a request.")
	}

	var response publishToCatalogResponse
	_, err = client.Do(request, &response)
	if err != nil {
		return err
	}

	io.Logf("%s Release published: %s=%s\n", color.GreenCheck(),
		color.Blue("url"), response.CatalogUrl)

	return nil
}

func publishToCatalogRequestBody(version string) (*publishToCatalogRequest, error) {
	baseDir, err := os.Getwd()
	if err != nil {
		return nil, cmdutils.WrapError(err, "failed to get working directory")
	}

	components, err := fetchTemplates(baseDir)
	if err != nil {
		return nil, cmdutils.WrapError(err, "failed to fetch components")
	}

	metadata := make(map[string]interface{})
	componentsData := make([]map[string]interface{}, 0, len(components))

	for name, path := range components {
		spec, err := extractSpec(path)
		if err != nil {
			return nil, cmdutils.WrapError(err, "failed to extract spec")
		}

		componentsData = append(componentsData, map[string]interface{}{
			"name":           name,
			"spec":           spec,
			"component_type": "template",
		})
	}

	sort.Slice(componentsData, func(i, j int) bool {
		return componentsData[i]["name"].(string) < componentsData[j]["name"].(string)
	})

	metadata["components"] = componentsData

	return &publishToCatalogRequest{
		Version:  version,
		Metadata: metadata,
	}, nil
}

// fetchTemplates returns a map of component names to their paths.
// The component name is either the name of the file without the extension in the "templates" directory of the project
// or the name of the directory containing a "template.yml" file in the "templates" directory.
// More information: https://docs.gitlab.com/ee/ci/components/index.html#directory-structure
func fetchTemplates(baseDir string) (map[string]string, error) {
	templates := make(map[string]string)

	paths, err := fetchTemplatePaths(baseDir)
	if err != nil {
		return nil, cmdutils.WrapError(err, "failed to fetch template paths")
	}

	for _, path := range paths {
		componentName, err := extractComponentName(baseDir, path)
		if err != nil {
			return nil, cmdutils.WrapError(err, "failed to extract component name")
		}

		if componentName != "" {
			templates[componentName] = path
		}
	}

	return templates, nil
}

// fetchTemplatePaths returns a list of the possible component paths to the YAML files in the "templates" directory.
func fetchTemplatePaths(baseDir string) ([]string, error) {
	templatesDir := filepath.Join(baseDir, templatesDirName)

	var yamlFiles []string

	err := filepath.WalkDir(templatesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return cmdutils.WrapError(err, "failed to walk directory")
		}

		if filepath.Ext(d.Name()) == templateFileExt {
			yamlFiles = append(yamlFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return yamlFiles, nil
}

// extractComponentName returns the valid component name from the path if it is a valid component path.
// valid component paths:
// 1. All YAML files in the "templates" directory.
// 2. All "template.yml" files in the subdirectories of the "templates" directory.
func extractComponentName(baseDir string, path string) (string, error) {
	relativePath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return "", err
	}

	dirname := filepath.Dir(relativePath)
	fileExt := filepath.Ext(relativePath)
	filename := filepath.Base(relativePath)
	filenameWithoutExt := filename[:len(filename)-len(fileExt)]

	// All YAML files in the "templates" directory.
	if dirname == templatesDirName {
		return filenameWithoutExt, nil
	}

	// All "template.yml" files in the subdirectories of the "templates" directory.
	if filename == templateFileName {
		return filepath.Base(dirname), nil
	} else {
		return "", nil
	}
}

type specDef struct {
	Spec map[string]interface{} `yaml:"spec"`
}

// extractSpec returns the spec from the component file.
func extractSpec(componentPath string) (map[string]interface{}, error) {
	content, err := os.ReadFile(componentPath)
	if err != nil {
		return nil, cmdutils.WrapError(err, "failed to read file")
	}

	var spec specDef

	err = yaml.Unmarshal(content, &spec)
	if err != nil {
		return nil, cmdutils.WrapError(err, "failed to unmarshal YAML")
	}

	return spec.Spec, nil
}
