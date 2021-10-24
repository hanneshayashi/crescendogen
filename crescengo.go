package crescengo

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var psProtectedVariables = []string{
	"host",
}

// OutputHandler is an OutputHandler in a Crescendo definition
type OutputHandler struct {
	ParameterSetName string `json:"ParameterSetName,omitempty"`
	Handler          string `json:"Handler,omitempty"`
	StreamOutput     bool   `json:"StreamOutput,omitempty"`
}

// Parameter represents a parameter definition for a PowerShell function in Crescendo format
type Parameter struct {
	Name          string `json:"Name,omitempty"`
	OriginalName  string `json:"OriginalName,omitempty"`
	ParameterType string `json:"ParameterType,omitempty"`
	Description   string `json:"Description,omitempty"`
	Mandatory     bool   `json:"Mandatory,omitempty"`
}

// CrescendoDef represents a single function definition for Crescendo
type CrescendoDef struct {
	Schema                  string          `json:"$Schema,omitempty"`
	Verb                    string          `json:"Verb,omitempty"`
	Noun                    string          `json:"Noun,omitempty"`
	OriginalName            string          `json:"OriginalName,omitempty"`
	OriginalCommandElements []string        `json:"OriginalCommandElements,omitempty"`
	OutputHandlers          []OutputHandler `json:"OutputHandlers,omitempty"`
	Description             string          `json:"Description,omitempty"`
	Parameters              []Parameter     `json:"Parameters,omitempty"`
}

// capFirstLetter capitalizes the first letter of a string
func capFirstLetter(s string) string {
	return strings.ToUpper(string(s[0])) + string(s[1:])
}

// contains checks if a string is inside a slice
func contains(s string, slice []string) bool {
	for i := range slice {
		if s == slice[i] {
			return true
		}
	}
	return false
}

func createCrescendoModuleDefs(commands []*cobra.Command, root, path string, defaultFlags ...string) {
	for _, command := range commands {
		if command.Use == "help [command]" {
			continue
		}
		children := command.Commands()
		if len(children) > 0 {
			createCrescendoModuleDefs(children, root, path, defaultFlags...)
		}
		cresDef := CrescendoDef{
			Schema:                  "./Microsoft.PowerShell.Crescendo.Schema.json",
			Description:             command.Short,
			OriginalName:            root,
			OriginalCommandElements: append(strings.Split(command.CommandPath(), " ")[1:], defaultFlags...),
			OutputHandlers: []OutputHandler{
				{
					ParameterSetName: "Default",
				},
			},
		}
		if command.Annotations["crescendoOutput"] != "" {
			cresDef.OutputHandlers[0].Handler = command.Annotations["crescendoOutput"]
		} else {
			cresDef.OutputHandlers[0].Handler = "$_ | ConvertFrom-Json"
			cresDef.OutputHandlers[0].StreamOutput = true
		}
		if command.Annotations["crescendoAttachToParent"] == "true" {
			cresDef.Verb = capFirstLetter(command.Parent().Use)
			cresDef.Noun = capFirstLetter(command.Parent().Parent().Use) + capFirstLetter(command.Use)
		} else {
			cresDef.Verb = capFirstLetter(command.Use)
			cresDef.Noun = capFirstLetter(command.Parent().Use)
		}
		foo := func(f *pflag.Flag) {
			if f.Name == "help" {
				return
			}
			originalName := "--" + f.Name
			if contains(originalName, defaultFlags) {
				return
			}
			p := Parameter{
				Name:         strings.ToUpper(string(f.Name[0])) + string(f.Name[1:]),
				OriginalName: originalName,
				Description:  f.Usage,
			}
			if contains(f.Name, psProtectedVariables) {
				p.Name += "_"
			}
			switch f.Value.Type() {
			case "bool":
				p.ParameterType = "switch"
			default:
				p.ParameterType = "string"
			}
			required, ok := f.Annotations[cobra.BashCompOneRequiredFlag]
			if ok && required[0] == "true" {
				p.Mandatory = true
			}
			cresDef.Parameters = append(cresDef.Parameters, p)
		}
		command.Flags().VisitAll(foo)
		command.Root().PersistentFlags().VisitAll(foo)
		fileName := strings.Join(cresDef.OriginalCommandElements[:len(cresDef.OriginalCommandElements)-len(defaultFlags)], "_")
		if command.Annotations["crescendoFlags"] != "" {
			cresDef.OriginalCommandElements = append(cresDef.OriginalCommandElements, command.Annotations["crescendoFlags"])
		}
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
		filePath := path + fileName + ".json"
		file, _ := json.MarshalIndent(cresDef, "", "\t")
		_ = ioutil.WriteFile(filePath, file, 0644)
	}
}

// CreateCrescendoModuleDefs creates module definitions in .json format for Microsoft Crescendo
// https://github.com/PowerShell/Crescendo
func CreateCrescendoModuleDefs(cmd *cobra.Command, path string, defaultFlags ...string) {
	commands := cmd.Commands()
	for _, command := range commands {
		children := command.Commands()
		if len(children) > 0 {
			createCrescendoModuleDefs(children, cmd.Use, path, defaultFlags...)
		}
	}
}
