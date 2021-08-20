package bigbang

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	ppath "path"
	"strings"

	// "github.com/karuppiah7890/go-jsonschema-generator"
	"github.com/runyontr/helm-schema-gen/pkg/document"
	"github.com/runyontr/helm-schema-gen/pkg/jsonschema"

	yamlv2 "gopkg.in/yaml.v2"
	yamlv3 "gopkg.in/yaml.v3"

	"github.com/norwoodj/helm-docs/pkg/helm"
)

func printNode(n *yamlv3.Node) {
	if n.Value != "" {
		fmt.Printf("---------------\n")
		fmt.Printf("Node Value: %v \n", n.Value)
		fmt.Printf("Node HeadComment: %v \n", n.HeadComment)
		fmt.Printf("Node LineComment: %v \n", n.LineComment)
		fmt.Printf("Node FootComment: %v \n", n.FootComment)
		fmt.Printf("---------------\n")
	}
	for _, subNode := range n.Content {
		printNode(subNode)
	}
}

// Run pulls the values file for the provivded bigbang version and then
// iterates through the appropriate packages to pull the values file for each
// package.

// It then merges the values file in an appropriate manner to create an
// JSON Schema file
func Run(path string) error {
	// https://repo1.dso.mil/platform-one/big-bang/bigbang/-/raw/master/chart/values.yaml
	// bb, err := get("https://repo1.dso.mil/platform-one/big-bang/bigbang", version)
	// if err != nil {
	// 	return err
	// }

	var fullChartSearchRoot string

	if ppath.IsAbs(path) {
		fullChartSearchRoot = path
	} else {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting working directory: %s\n", err)
			return err
		}

		fullChartSearchRoot = ppath.Join(cwd, path)
	}

	chartDocumentationInfo, err := helm.ParseChartInformation(fullChartSearchRoot)
	if err != nil {
		fmt.Printf("Error!!! %v\n", err)
	}

	if err != nil {
		fmt.Printf("Error parsing information for chart %s, skipping: %s", path, err)
		return err
	}
	// b, _ := json.MarshalIndent(chartDocumentationInfo, "", "\t")
	// fmt.Printf("%v\n", string(b))

	// Parse the yaml node
	// printNode(chartDocumentationInfo.ChartValues)

	// // document.PrintDocumentation(chartDocumentationInfo, fullChartSearchRoot, []string{"README.md.gotmpl"}, true, "1.5.0")
	bbValues, err := ioutil.ReadFile(fullChartSearchRoot + "/values.yaml")
	if err != nil {
		fmt.Printf("Error reading values file: %v\n", err)
		return err
	}
	// Parse the http body into a yaml structure
	// TODO pull out comments!
	values := make(map[string]interface{})

	err = yamlv2.Unmarshal(bbValues, &values)
	// s := &jsonschema.Document{}
	// s.ReadDeep(&values)
	// fmt.Println(s)

	//Istio...
	s := make(map[string]map[string]map[string]interface{})
	err = yamlv2.Unmarshal(bbValues, &s)
	nested := make(map[string]map[string]interface{})
	err = yamlv2.Unmarshal(bbValues, &nested)
	// Should be able to loop smartly on paths with git
	for p, top := range s {
		for key, val := range top {
			if key == "git" {
				// found a git!
				repo := val["repo"].(string)
				tag := val["tag"].(string)
				// fmt.Printf("Getting %v@%v\n", repo, tag)
				b, err := get(repo, tag)
				if err != nil {
					// fmt.Printf("Error getting values from %v@%v: %v", repo, tag, err)
					return err
				}
				// fmt.Printf(string(ib))
				subValues := make(map[string]interface{})
				err = yamlv2.Unmarshal(b, &subValues)

				nested[p]["values"] = subValues
				// fmt.Printf("Added Values to %v\n", p)
			}
		}
	}

	//addons
	addons := make(map[string]map[string]map[string]map[string]interface{})
	err = yamlv2.Unmarshal(bbValues, &addons)
	for p, top := range addons["addons"] {
		for key, val := range top {
			if key == "git" {
				// found a git!
				// fmt.Printf("addons.%v.%v\n", p, key)
				repo := val["repo"].(string)
				tag := val["tag"].(string)
				// fmt.Printf("Getting %v@%v\n", repo, tag)
				b, err := get(repo, tag)
				if err != nil {
					// fmt.Printf("Error getting values from %v@%v: %v", repo, tag, err)
					return err
				}
				// fmt.Printf(string(ib))
				subValues := make(map[string]interface{})
				err = yamlv2.Unmarshal(b, &subValues)

				nested["addons"][p].(map[interface{}]interface{})["values"] = subValues
				// fmt.Printf("Added Values to %v\n", p)
			}
		}
	}
	// tag := strings["istio"]["git"]["tag"]
	// repo := strings["istio"]["git"]["repo"]
	// // istioVersion := values["istio"]["git"]["tag"]
	// // fmt.Printf("Istio Version: %v\n", tag)
	// // fmt.Println(string(b))

	// ib, err := get(repo.(string), tag.(string))
	// if err != nil {
	// 	return err
	// }
	// // fmt.Printf(string(ib))
	// istioValues := make(map[string]interface{})
	// err = yaml.Unmarshal(ib, &istioValues)

	// fmt.Println(i)

	//we want the top level istio properties injected into the
	// bigbang properties at .istio.values.properties....

	nestedDocument := &jsonschema.Document{}
	nestedDocument.ReadDeep(&nested)

	// supplementJSONSchema(chartDocumentationInfo.ChartValues, nestedDocument, ".", true)

	// fmt.Println(nestedDocument)

	valuesTableRows, err := document.CreateValueRowsFromField(
		"",
		nil,
		chartDocumentationInfo.ChartValues.Content[0],
		chartDocumentationInfo.ChartValuesDescriptions,
		true,
	)

	if err != nil {
		fmt.Printf("Error CreatingValueRoes from Field: %v\n", err)
		return err
	}
	var p *jsonschema.Property
	for _, row := range valuesTableRows {
		// fmt.Printf("%+v\n", row)
		// see the path in json
		p = &nestedDocument.Property

		// fmt.Printf("Top Level JSONSchema Doc:")
		// b, _ = json.MarshalIndent(p, row.Key, "\t")
		// fmt.Printf("%v\n", string(b))
		for _, k := range strings.Split(row.Key, ".") {
			// fmt.Printf("%v: %v\n", i, k)
			p = p.Properties[k]
			// doc = &jsonschema.Document{
			// 	Property: *doc.Properties[k],
			// }

			// b, _ = json.MarshalIndent(p, k, "\t")
			// fmt.Printf("%v\n", string(b))
		}
		if p == nil {
			continue
		}
		p.Description = row.AutoDescription
		p.Default = strings.TrimSuffix(strings.TrimPrefix(row.Default, "`"), "`")

		// fmt.Printf("Intrim STATE")
		// b, _ = json.MarshalIndent(p, "", "\t")
		// fmt.Printf("%v\n", string(b))
	}

	b, _ := json.MarshalIndent(nestedDocument, "", "\t")
	fmt.Printf("%v\n", string(b))
	return nil
}

func supplementJSONSchema(n *yamlv3.Node, document *jsonschema.Document, parent string, isRoot bool) {
	if isRoot {
		for _, subNode := range n.Content {
			supplementJSONSchema(subNode, document, parent, false)
		}
		return
	}
	fmt.Printf("----New Node----\n")
	fmt.Printf(parent+" Node Kind: %v \n", kindToString(n.Kind))
	fmt.Printf(parent+" Node Style: %v \n", n.Style)
	fmt.Printf(parent+" Node Tag: %v \n", n.ShortTag())
	fmt.Printf(parent+" Node Value: %v \n", n.Value)
	fmt.Printf(parent+" Node Anchor: %v \n", n.Anchor)
	fmt.Printf(parent+" Node Alias: %v \n", n.Alias)
	fmt.Printf(parent+" Node Line: %v \n", n.Line)
	fmt.Printf(parent+" Node Col: %v \n", n.Column)
	// fmt.Printf("New Node:\nKind: %v\nAnchor: %v\nTag: %v\n", n.Kind, n.Anchor, n.Tag)
	if n.Value != "" {
		fmt.Printf("---------------\n")
		fmt.Printf("Node Value: %v \n", n.Value)
		// fmt.Printf("Node HeadComment: %v \n", n.HeadComment)
		// fmt.Printf("Node LineComment: %v \n", n.LineComment)
		// fmt.Printf("Node FootComment: %v \n", n.FootComment)
		fmt.Printf("---------------\n")
		document.Description = n.HeadComment
	} else {
		fmt.Printf("----No Value-----\n")
	}

	if len(n.Content) == 0 {
		if n.Content != nil {
			fmt.Printf("Node %v is of type %v with defautl value %v\n", n.Value, n.Content[0].Tag, n.Content[0].Value)
		}
	}

	if n.Tag == "!!map" {
		fmt.Printf("Node %v is a map\n", n.Value)
	} else if n.Tag == "!!str" {
		fmt.Printf("Node %v is a string\n", n.Value)
		out := ""
		n.Decode(&out)
		fmt.Printf("String value for node %v: %v\n", n.Value, out)
	}
	fmt.Printf("Node %v has %v children\n", n.Value, len(n.Content))
	for _, subNode := range n.Content {
		fmt.Printf("Child Node: %v\n", subNode.Value)
	}
	if len(n.Content) == 1 {
		fmt.Printf("Assuming node %v has default value %v\n", n.Value, n.Content[0].Value)
	} else {
		for _, subNode := range n.Content {
			supplementJSONSchema(subNode, document, parent+n.Value, false)
		}
	}
}

func kindToString(kind yamlv3.Kind) string {
	switch kind {
	case yamlv3.DocumentNode:
		return "DocumentNode"
	case yamlv3.SequenceNode:
		return "SequenceNode"
	case yamlv3.MappingNode:
		return "MappingNode"
	case yamlv3.ScalarNode:
		return "ScalarNode"
	case yamlv3.AliasNode:
		return "AliasNode"
	}
	return "unknown"
}

func get(baseURL, version string) ([]byte, error) {
	// https://repo1.dso.mil/platform-one/big-bang/bigbang/-/raw/master/chart/values.yaml
	// Some repos have a .git, so we trim that.
	bbUrl := fmt.Sprintf("%v/-/raw/%v/chart/values.yaml", strings.TrimSuffix(baseURL, ".git"), version)

	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go

	// curl https://repo1.dso.mil/platform-one/big-bang/bigbang/-/raw/master/chart/values.yaml

	resp, err := http.Get(bbUrl)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
