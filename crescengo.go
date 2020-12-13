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
}

// Parameter represents a parameter definition for a PowerShell function in Crescendo format
type Parameter struct {
	Name          string `json:"Name,omitempty"`
	OriginalName  string `json:"OriginalName,omitempty"`
	ParameterType string `json:"ParameterType,omitempty"`
	Description   string `json:"Description,omitempty"`
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

func createCrescendoModuleDefs(commands []*cobra.Command, root, path string) {
	for _, command := range commands {
		children := command.Commands()
		if len(children) > 0 {
			createCrescendoModuleDefs(children, root, path)
		}
		cresDef := CrescendoDef{
			Schema:                  "./Microsoft.PowerShell.Crescendo.Schema.json",
			Description:             command.Short,
			OriginalName:            root,
			OriginalCommandElements: append(strings.Split(command.CommandPath(), " ")[1:], "--compressOutput"),
			OutputHandlers: []OutputHandler{
				{
					ParameterSetName: "Default",
				},
			},
		}
		if command.Annotations["crescendoOutput"] != "" {
			cresDef.OutputHandlers[0].Handler = command.Annotations["crescendoOutput"]
		} else {
			cresDef.OutputHandlers[0].Handler = "$args[0] | ConvertFrom-Json"
		}
		if command.Annotations["crescendoAttachToParent"] == "true" {
			cresDef.Verb = capFirstLetter(command.Parent().Use)
			cresDef.Noun = capFirstLetter(command.Parent().Parent().Use) + capFirstLetter(command.Use)
		} else {
			cresDef.Verb = capFirstLetter(command.Use)
			cresDef.Noun = capFirstLetter(command.Parent().Use)
		}
		command.Flags().VisitAll(func(f *pflag.Flag) {
			p := Parameter{
				Name:         strings.ToUpper(string(f.Name[0])) + string(f.Name[1:]),
				OriginalName: "--" + f.Name,
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
			cresDef.Parameters = append(cresDef.Parameters, p)
		})
		fileName := strings.Join(cresDef.OriginalCommandElements[:len(cresDef.OriginalCommandElements)-1], "_")
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
func CreateCrescendoModuleDefs(cmd *cobra.Command, path string) {
	commands := cmd.Commands()
	for _, command := range commands {
		children := command.Commands()
		if len(children) > 0 {
			createCrescendoModuleDefs(children, cmd.Use, path)
		}
	}
}
